// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package nonodo

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/readerclient"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

const testTimeout = 100 * time.Second

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

//
// Test Cases
//

func (s *NonodoSuite) TestRead2() {
	opts := NewNonodoOpts()
	// we are using file cuz there is problem with memory
	// no such table: reports
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	opts.EnableEcho = true
	opts.SqliteFile = path.Join(tempDir, sqliteFileName)
	s.setupTest(opts)
	defer os.RemoveAll(tempDir)
	s.T().Log("sending advance inputs")
	const n = 3
	var payloads [n][32]byte
	for i := 0; i < n; i++ {
		payloads[i] = s.makePayload()
		err := devnet.AddInput(s.ctx, s.rpcUrl, payloads[i][:])
		s.NoError(err)
	}

	s.T().Log("waiting until last input is ready")
	err = s.waitForAdvanceInput(n - 1)
	s.NoError(err)

	client, err := ethclient.DialContext(s.ctx, fmt.Sprintf("http://127.0.0.1:8546"))
	appAddress := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	inputBoxAddress := common.HexToAddress("0x58Df21fE097d4bE5dCf61e01d9ea3f6B81c2E1dB")
	inputBox, err := contracts.NewInputBox(appAddress, client)
	l1FinalizedPrevHeight := uint64(1)
	timestamp := uint64(1727126680000)
	s.NoError(err)

	w := inputter.InputterWorker{
		Model:              nil,
		Provider:           "",
		InputBoxAddress:    appAddress,
		InputBoxBlock:      1,
		ApplicationAddress: inputBoxAddress,
	}
	lastL1BlockRead, err := w.ReadInputsByBlockAndTimestamp(s.ctx, client, inputBox, l1FinalizedPrevHeight, timestamp-5000)
	s.NoError(err)
	s.NotNil(lastL1BlockRead)

}

//
// Setup and tear down
//

// Setup the nonodo suite.
// This method requires the nonodo options, so each test must call it explicitly.
func (s *NonodoSuite) setupTest(opts NonodoOpts) {
	s.nonce += 1
	opts.AnvilPort += s.nonce
	opts.HttpPort += s.nonce + 100
	s.T().Log("ports", "http", opts.HttpPort, "anvil", opts.AnvilPort)
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	var workerCtx context.Context
	workerCtx, s.workerCancel = context.WithCancel(s.ctx)

	w := NewSupervisor(opts)

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
		result, err := readerclient.InputStatus(s.ctx, s.graphqlClient, inputIndex)
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
