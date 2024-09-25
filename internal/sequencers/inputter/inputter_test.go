// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package inputter

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

type InputterTestSuite struct {
	suite.Suite
}

func (s *InputterTestSuite) TestReadInputsByBlockAndTimestamp() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, "http://127.0.0.1:8545")
	s.NoError(err)
	appAddress := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	inputBoxAddress := common.HexToAddress("0x58Df21fE097d4bE5dCf61e01d9ea3f6B81c2E1dB")
	inputBox, err := contracts.NewInputBox(appAddress, client)
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
	lastL1BlockRead, err := w.ReadInputsByBlockAndTimestamp(ctx, client, inputBox, l1FinalizedPrevHeight, timestamp-5000)
	s.NoError(err)
	s.NotNil(lastL1BlockRead)
}

func TestInputterTestSuite(t *testing.T) {
	suite.Run(t, &InputterTestSuite{})
}
