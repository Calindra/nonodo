// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package nonodo

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/calindra/cartesi-rollups-graphql/pkg/convenience"
	"github.com/calindra/cartesi-rollups-graphql/pkg/reader"
	"github.com/calindra/cartesi-rollups-graphql/pkg/readerclient"
	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 5 * time.Second

type NonodoSuite struct {
	suite.Suite
	ctx           context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
	graphqlClient graphql.Client
	inspectClient *inspect.ClientWithResponses
	nonce         int
}

// Mini implementation of GraphQL using hl like library
type InlineHLGraphQL struct {
	w supervisor.HttpWorker
}

// Start implements supervisor.Worker.
func (i InlineHLGraphQL) Start(ctx context.Context, ready chan<- struct{}) error {
	return i.w.Start(ctx, ready)
}

// String implements supervisor.Worker.
func (i InlineHLGraphQL) String() string {
	return "InlineHLGraphQL"
}

func NewInlineHLGraphQLWorker(opts NonodoOpts) supervisor.Worker {
	db := CreateDBInstance(opts)
	container := convenience.NewContainer(*db, opts.AutoCount)
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      opts.TimeoutInspect,
	}))
	reader.Register(e, convenienceService, adapter)

	return InlineHLGraphQL{
		w: supervisor.HttpWorker{
			Address: fmt.Sprintf("%s:%d", opts.HttpAddress, opts.AnvilPort),
			Handler: e,
		},
	}
}

//
// Test Cases
//

func (s *NonodoSuite) TestItProcessesAdvanceInputs() {
	opts := NewNonodoOpts()
	// we are using file cuz there is problem with memory
	// no such table: reports
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	opts.EnableEcho = true
	opts.SqliteFile = filepath.Join(tempDir, sqliteFileName)
	s.setupTest(&opts)
	defer os.RemoveAll(tempDir)
	s.T().Log("sending advance inputs")
	const n = 3
	var payloads [n][32]byte
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := devnet.AddInput(s.ctx, s.rpcUrl, payloads[i][:])
		s.Require().NoError(err)
	}

	s.T().Log("waiting until last input is ready")
	err = s.waitForAdvanceInput(n - 1)
	s.NoError(err)

	s.T().Log("inserting proofs")
	db := CreateDBInstance(opts)
	_, err = db.Exec(`update vouchers set output_hashes_siblings = $1`, `["0x11","0x22","0x33"]`)
	s.NoError(err)
	_, err = db.Exec(`update notices set output_hashes_siblings = $1`, `["0x1111","0x2222","0x3333"]`)
	s.NoError(err)

	s.T().Log("verifying node state")
	response, err := readerclient.State(s.ctx, s.graphqlClient)
	s.Require().NoError(err)
	for i := 0; i < n; i++ {
		input := response.Inputs.Edges[i].Node
		s.Equal(i, input.Index)
		s.Equal(payloads[i][:], s.decodeHex(input.Payload))
		s.Equal(devnet.SenderAddress, input.MsgSender)

		// check voucher
		voucher := input.Vouchers.Edges[0].Node
		s.Equal(model.VOUCHER_SELECTOR, voucher.Payload[2:10])
		s.True(strings.HasSuffix(voucher.Payload, common.Bytes2Hex(payloads[i][:]))) // should ends with
		s.Equal(devnet.SenderAddress, voucher.Destination)
		s.Equal(3, len(voucher.Proof.OutputHashesSiblings))

		// check notice
		notice := input.Notices.Edges[0].Node
		s.Equal(model.NOTICE_SELECTOR, notice.Payload[2:10])
		s.Contains(notice.Payload, common.Bytes2Hex(payloads[i][:])+"ff")
		s.Equal(3, len(notice.Proof.OutputHashesSiblings))

		// check report
		s.Equal(payloads[i][:], s.decodeHex(input.Reports.Edges[0].Node.Payload))
	}

	s.T().Log("query graphql state with new graphql path pattern (no results)")
	graphqlEndpoint := fmt.Sprintf("http://%s:%v/graphql/%s", opts.HttpAddress, opts.HttpPort, common.Address{}.Hex())
	graphqlClient := graphql.NewClient(graphqlEndpoint, nil)
	response2, err := readerclient.State(s.ctx, graphqlClient)
	s.NoError(err)
	s.Equal(0, len(response2.Inputs.Edges))

	s.T().Log("query graphql state with new graphql path pattern")
	graphqlEndpoint3 := fmt.Sprintf("http://%s:%v/graphql/%s", opts.HttpAddress, opts.HttpPort, devnet.ApplicationAddress)
	graphqlClient3 := graphql.NewClient(graphqlEndpoint3, nil)
	response3, err := readerclient.State(s.ctx, graphqlClient3)
	s.NoError(err)
	s.Equal(3, len(response3.Inputs.Edges))
}

func (s *NonodoSuite) TestItProcessesInspectInputs() {
	opts := NewNonodoOpts()
	opts.EnableEcho = true
	s.setupTest(&opts)

	s.T().Log("sending inspect inputs")
	const n = 3
	for i := 0; i < n; i++ {
		payload := s.makePayload()
		response, err := s.sendInspect(payload[:])
		s.NoError(err)
		s.Equal(http.StatusOK, response.StatusCode())
		s.Equal("0x", response.JSON200.ExceptionPayload)
		s.Equal(0, response.JSON200.ProcessedInputCount)
		s.Len(response.JSON200.Reports, 1)
		s.Equal(payload[:], s.decodeHex(response.JSON200.Reports[0].Payload))
		s.Equal(inspect.Accepted, response.JSON200.Status)
	}
}

func (s *NonodoSuite) TestItWorksWithExternalApplication() {
	opts := NewNonodoOpts()
	opts.ApplicationArgs = []string{
		"go",
		"run",
		"github.com/calindra/nonodo/internal/echoapp/echoapp",
		"--endpoint",
		fmt.Sprintf("http://%v:%v", opts.HttpAddress, opts.HttpRollupsPort),
	}
	opts.HttpPort = 8090
	s.setupTest(&opts)
	time.Sleep(100 * time.Millisecond)

	s.T().Log("sending inspect to external application")
	payload := s.makePayload()

	response, err := s.sendInspect(payload[:])
	s.NoError(err)
	slog.Debug("response", "body", string(response.Body))
	s.Require().Equal(http.StatusOK, response.StatusCode())
	s.Require().Equal(payload[:], s.decodeHex(response.JSON200.Reports[0].Payload))
}

//
// Setup and tear down
//

// Setup the nonodo suite.
// This method requires the nonodo options, so each test must call it explicitly.
func (s *NonodoSuite) setupTest(opts *NonodoOpts) {
	s.nonce += 1
	opts.AnvilPort += s.nonce
	opts.HttpPort += s.nonce + 100
	opts.AnvilCommand = "anvil"
	s.T().Log("ports", "http", opts.HttpPort, "anvil", opts.AnvilPort)
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	var workerCtx context.Context
	workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	w := NewSupervisor(*opts)
	// gqlw := NewInlineHLGraphQLWorker(*opts)
	// w.Workers = append([]supervisor.Worker{gqlw}, w.Workers...)

	ready := make(chan struct{})
	go func() {
		s.workerResult <- w.Start(workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("nonodo ready")
	}

	s.rpcUrl = fmt.Sprintf("http://127.0.0.1:%v", opts.AnvilPort)

	graphqlEndpoint := fmt.Sprintf("http://%s:%v/graphql", opts.HttpAddress, opts.HttpPort)
	s.graphqlClient = graphql.NewClient(graphqlEndpoint, nil)

	inspectEndpoint := fmt.Sprintf("http://%s:%v/", opts.HttpAddress, opts.HttpPort)
	var err error
	s.inspectClient, err = inspect.NewClientWithResponses(inspectEndpoint)
	s.NoError(err)
}

func (s *NonodoSuite) TearDownTest() {
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	s.T().Log("teardown ok.")
}

//
// Helper functions
//

// Wait for the given input to be ready.
func (s *NonodoSuite) waitForAdvanceInput(inputIndex int) error {
	const pollRetries = 100
	const pollInterval = 15 * time.Millisecond
	time.Sleep(100 * time.Millisecond)
	for i := 0; i < pollRetries; i++ {
		result, err := readerclient.InputStatus(s.ctx, s.graphqlClient, strconv.Itoa(inputIndex))
		if err != nil && !strings.Contains(err.Error(), "input not found") {
			return fmt.Errorf("failed to get input status: %w", err)
		}
		if result.Input.Status == readerclient.CompletionStatusAccepted {
			return nil
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("input never got ready")
}

// Create a random payload to use in the tests
func (s *NonodoSuite) makePayload() [32]byte {
	var payload [32]byte
	_, err := rand.Read(payload[:])
	s.NoError(err)
	fmt.Println(payload)
	return payload
}

// Decode the hex string into bytes.
func (s *NonodoSuite) decodeHex(value string) []byte {
	bytes, err := hexutil.Decode(value)
	s.NoError(err)
	return bytes
}

// Send an inspect request with the given payload.
func (s *NonodoSuite) sendInspect(payload []byte) (*inspect.InspectPostResponse, error) {
	return s.inspectClient.InspectPostWithBodyWithResponse(
		s.ctx,
		devnet.ApplicationAddress,
		"application/octet-stream",
		bytes.NewReader(payload),
	)
}

//
// Suite entry point
//

func TestNonodoSuite(t *testing.T) {
	suite.Run(t, &NonodoSuite{nonce: 0})
}
