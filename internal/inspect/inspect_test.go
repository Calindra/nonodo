// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package inspect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const TestTimeout = 5 * time.Second

// Model mock //////////////////////////////////////////////////////////////////////////////////////

type ModelMock struct {
	mock.Mock
}

func (m *ModelMock) AddInspectInput(payload []byte) int {
	args := m.Called(payload)
	return args.Int(0)
}

func (m *ModelMock) GetInspectInput(index int) model.InspectInput {
	args := m.Called(index)
	return args.Get(0).(model.InspectInput)
}

// setInspectInput sets the model to wait for the given inspect input payload, and returns the
// expected inspected result.
func (m *ModelMock) setInspectInput(payload []byte) {
	m.On("AddInspectInput", payload).Return(0)
	m.On("GetInspectInput", 0).Return(model.InspectInput{
		Index:               0,
		Status:              model.CompletionStatusAccepted,
		Payload:             payload,
		ProcessedInputCount: 0,
		Reports: []model.Report{
			{Payload: payload},
		},
		Exception: nil,
	})
}

// Test suite setup ////////////////////////////////////////////////////////////////////////////////

func TestInspectSuite(t *testing.T) {
	suite.Run(t, new(InspectSuite))
}

type InspectSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	model       *ModelMock
	server      *http.Server
	serveResult chan error
}

func (s *InspectSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), TestTimeout)
	s.model = &ModelMock{}

	router := echo.New()
	router.Use(middleware.Logger())
	router.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      100 * time.Millisecond,
	}))
	inspect := &inspectAPI{s.model}
	RegisterHandlers(router, inspect)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().Nil(err)
	slog.Info("listening", "address", ln.Addr())
	s.server = new(http.Server)
	s.server.Addr = ln.Addr().String()
	s.server.Handler = router
	s.serveResult = make(chan error, 1)
	go func() {
		s.serveResult <- s.server.Serve(ln)
	}()
}

func (s *InspectSuite) TearDownTest() {
	s.Nil(s.server.Shutdown(s.ctx))
	select {
	case serveResult := <-s.serveResult:
		s.Equal(serveResult, http.ErrServerClosed)
	case <-s.ctx.Done():
		s.T().Error(s.ctx.Err())
	}
	s.cancel()
}

// Test cases //////////////////////////////////////////////////////////////////////////////////////

func (s *InspectSuite) TestGetWithSimplePayload() {
	s.testGet("hello")
}

func (s *InspectSuite) TestGetWithSpaces() {
	s.testGet("hello world")
}

func (s *InspectSuite) TestGetWithUrlEncodedPayload() {
	s.testGet("hello%20world")
}

func (s *InspectSuite) TestGetWithSlashes() {
	s.testGet("user/123/name")
}

func (s *InspectSuite) TestGetWithPathAndQuery() {
	s.testGet("user/data?key=value&key2=value2")
}

func (s *InspectSuite) TestGetWithJson() {
	s.testGet(`{"key": ["value1", "value2"]}`)
}

func (s *InspectSuite) TestGetFailsWithEmptyPayload() {
	status, body := s.doGetInspect("")
	s.Equal(http.StatusNotFound, status)
	s.Contains(body, "Not Found")
}

func (s *InspectSuite) TestPostWithEmptyPayload() {
	s.testPost([]byte{})
}

func (s *InspectSuite) TestPostWithBinaryPayload() {
	s.testPost(common.Hex2Bytes("deadbeef"))
}

func (s *InspectSuite) TestPostWithPayloadOnSizeLimit() {
	s.testPost(make([]byte, PayloadSizeLimit))
}

func (s *InspectSuite) TestPostWithPayloadOverSizeLimit() {
	status, body := s.doPostInspect(make([]byte, PayloadSizeLimit+1))
	s.Equal(http.StatusBadRequest, status)
	s.Contains(body, "Payload reached size limit")
}

func (s *InspectSuite) TestRequestFailsWithTimeout() {
	s.model.On("AddInspectInput", []byte{}).Return(0)
	s.model.On("GetInspectInput", 0).Return(model.InspectInput{
		Index:  0,
		Status: model.CompletionStatusUnprocessed,
	})
	status, body := s.doPostInspect([]byte{})
	s.Equal(http.StatusServiceUnavailable, status)
	s.Contains(body, "Request timed out")
}

// Helper functions ////////////////////////////////////////////////////////////////////////////////

func (s *InspectSuite) doHttpRequest(req *http.Request) (int, string) {
	resp, err := http.DefaultClient.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	s.Require().Nil(err)
	return resp.StatusCode, string(body)
}

func (s *InspectSuite) doGetInspect(payload string) (int, string) {
	url := fmt.Sprintf("http://%v/inspect/%v", s.server.Addr, payload)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, url, nil)
	s.Require().Nil(err)
	return s.doHttpRequest(req)
}

func (s *InspectSuite) doPostInspect(payload []byte) (int, string) {
	url := fmt.Sprintf("http://%v/inspect", s.server.Addr)
	req, err := http.NewRequestWithContext(
		s.ctx, http.MethodPost, url, bytes.NewReader(payload))
	s.Require().Nil(err)
	return s.doHttpRequest(req)
}

func (s *InspectSuite) testGet(escapedPayload string) {
	unescaped, err := url.QueryUnescape(escapedPayload)
	s.Require().Nil(err)
	payload := []byte(unescaped)
	s.model.setInspectInput(payload)
	status, body := s.doGetInspect(escapedPayload)
	s.Equal(http.StatusOK, status)
	s.Equal(echoResult(payload), body)
}

func (s *InspectSuite) testPost(payload []byte) {
	s.model.setInspectInput(payload)
	status, body := s.doPostInspect(payload)
	s.Equal(http.StatusOK, status)
	s.Equal(echoResult(payload), body)
}

// echoResult returns the expected JSON result of an echo application.
func echoResult(payload []byte) string {
	result := InspectResult{
		ExceptionPayload:    "0x",
		ProcessedInputCount: 0,
		Reports: []Report{
			{Payload: hexutil.Encode(payload)},
		},
		Status: Accepted,
	}
	body, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	return string(append(body, '\n'))
}
