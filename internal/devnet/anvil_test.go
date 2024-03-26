// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package devnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

const testTimeout = 5 * time.Second

func TestAnvilWorker(t *testing.T) {
	ctx, timeoutCancel := context.WithTimeout(context.Background(), testTimeout)
	defer timeoutCancel()

	w := AnvilWorker{
		Address: AnvilDefaultAddress,
		Port:    AnvilDefaultPort,
		Verbose: true,
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
		t.Error(ctx.Err())
	}

	// send input
	rpcUrl := fmt.Sprintf("http://127.0.0.1:%v", AnvilDefaultPort)
	payload := common.Hex2Bytes("deadbeef")
	err := AddInput(ctx, rpcUrl, payload)
	assert.Nil(t, err)

	// read input
	events, err := GetInputAdded(ctx, rpcUrl)
	assert.Nil(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, payload, events[0].Input)

	// stop worker
	workerCancel()
	select {
	case err := <-result:
		assert.Equal(t, context.Canceled, err)
	case <-ctx.Done():
		t.Error(ctx.Err())
	}
}
