// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
)

// The HTTP worker starts and manage an HTTP server.
type HttpWorker struct {
	Address string
	Handler http.Handler
}

func (w HttpWorker) String() string {
	return "http"
}

func (w HttpWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	// create the server
	server := http.Server{
		Addr:    w.Address,
		Handler: w.Handler,
	}
	ln, err := net.Listen("tcp", w.Address)
	if err != nil {
		return err
	}

	ready <- struct{}{}

	// create the goroutine to shutdown server
	go func() {
		<-ctx.Done()
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("http: error shutting down server", "error", err)
		}
	}()

	// serve
	err = server.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
