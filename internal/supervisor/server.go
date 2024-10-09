// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package supervisor

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Poll interval when checking whether the server is ready.
const ServerPollInterval = 10 * time.Millisecond

// This worker is responsible for a shell command that runs endlessly.
// The worker polls the given port to know when it is ready.
type ServerWorker struct {
	CommandWorker
	Port int
}

func (w ServerWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	commandReady := make(chan struct{})
	var d net.Dialer
	go func() {
		// Poll the TCP once the command is ready
		select {
		case <-ctx.Done():
			return
		case <-commandReady:
		}
		for {
			conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("0.0.0.0:%v", w.Port))
			if err == nil {
				conn.Close()
				ready <- struct{}{}
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(ServerPollInterval):
			}
		}
	}()
	return w.CommandWorker.Start(ctx, commandReady)
}
