// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This pkg is a binary for the echo application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/calindra/nonodo/internal/echoapp"
	"github.com/carlmjohnson/versioninfo"
	"github.com/spf13/cobra"
)

var cmd = &cobra.Command{
	Use:     "echoapp",
	Short:   "Echo application that uses the rollup HTTP API",
	Run:     run,
	Version: versioninfo.Short(),
}

var endpoint string

func init() {
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Rollup HTTP API endpoint")
	cobra.CheckErr(cmd.MarkFlagRequired("endpoint"))
}

func run(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	w := echoapp.EchoAppWorker{
		RollupEndpoint: endpoint,
	}
	ready := make(chan struct{})
	go func() {
		select {
		case <-ready:
			slog.Info("echo: application started")
		case <-ctx.Done():
		}
	}()
	err := w.Start(ctx, ready)
	if !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func main() {
	cobra.CheckErr(cmd.Execute())
}
