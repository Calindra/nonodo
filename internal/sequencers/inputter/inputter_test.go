// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package inputter

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

type InputterTestSuite struct {
	suite.Suite
	ctx           context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
	// nonce         int
}

func (s *InputterTestSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	slog.Debug("Setup!!!")
	var w supervisor.SupervisorWorker
	w.Name = "TesteInputter"
	s.ctx = context.Background()
	anvilLocation, err := devnet.CheckAnvilAndInstall(s.ctx)
	s.NoError(err)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:  devnet.AnvilDefaultAddress,
		Port:     devnet.AnvilDefaultPort,
		Verbose:  true,
		AnvilCmd: anvilLocation,
	})
	var workerCtx context.Context
	workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	s.rpcUrl = fmt.Sprintf("ws://%s:%v", devnet.AnvilDefaultAddress, devnet.AnvilDefaultPort)
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
}

func (s *InputterTestSuite) TestReadInputsByBlockAndTimestamp() {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, "http://127.0.0.1:8545")
	s.NoError(err)
	appAddress := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	inputBoxAddress := common.HexToAddress("0x58Df21fE097d4bE5dCf61e01d9ea3f6B81c2E1dB")
	inputBox, err := contracts.NewInputBox(inputBoxAddress, client)
	s.NoError(err)
	l1FinalizedPrevHeight := uint64(1)
	timestamp := uint64(1727126680000)
	w := InputterWorker{
		Model:              nil,
		Provider:           "",
		InputBoxAddress:    inputBoxAddress,
		InputBoxBlock:      1,
		ApplicationAddress: appAddress,
	}
	lastL1BlockRead, err := w.ReadInputsByBlockAndTimestamp(ctx, client, inputBox, l1FinalizedPrevHeight, (timestamp/1000)-300)
	s.NoError(err)
	s.NotNil(lastL1BlockRead)
}

func (s *InputterTestSuite) TearDownTest() {
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	err := exec.Command("pkill", "avail").Run()
	s.NoError(err)
	s.T().Log("teardown ok.")
}

func TestInputterTestSuite(t *testing.T) {
	suite.Run(t, &InputterTestSuite{})
}
