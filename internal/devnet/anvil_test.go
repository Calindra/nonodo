// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package devnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type AnvilSuite struct {
	suite.Suite
}

const testTimeout = 10 * time.Second

func (s *AnvilSuite) TestAnvilWorker() {
	ctx, timeoutCancel := context.WithTimeout(context.Background(), testTimeout)
	defer timeoutCancel()

	anvilPort := AnvilDefaultPort + 100
	w := AnvilWorker{
		Address:  AnvilDefaultAddress,
		Port:     anvilPort,
		Verbose:  true,
		AnvilCmd: "anvil",
	}

	// start worker in goroutine
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	ready := make(chan struct{})
	result := make(chan error)
	go func() {
		result <- w.Start(workerCtx, ready)
	}()

	// wait until worker is ready
	select {
	case <-ready:
	case <-ctx.Done():
		s.NoError(ctx.Err())
	}

	// send input
	rpcUrl := fmt.Sprintf("http://127.0.0.1:%v", anvilPort)
	payload := common.Hex2Bytes("deadbeef")
	err := AddInput(ctx, rpcUrl, payload)
	s.NoError(err)

	// read input
	events, err := GetInputAdded(ctx, rpcUrl)
	s.NoError(err)
	s.Equal(1, len(events))
	s.Equal(payload, events[0].Input)

	// stop worker
	workerCancel()
	canceled := false
	select {
	case err := <-result:
		s.Equal(context.Canceled, err)
		canceled = true
	case <-ctx.Done():
		s.NoError(ctx.Err())
	}
	s.True(canceled)
}

//
// Suite entry point
//

func TestAnvilSuite(t *testing.T) {
	suite.Run(t, &AnvilSuite{})
}
