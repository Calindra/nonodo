// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

//go:build !windows

package supervisor

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
)

// This worker is responsible for a shell command that runs endlessly.
type CommandWorker struct {
	Name    string
	Command string
	Args    []string
	Env     []string
}

func (w CommandWorker) String() string {
	return w.Name
}

func (w CommandWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	cmd := exec.CommandContext(ctx, w.Command, w.Args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, w.Env...)
	cmd.Stderr = &commandLogger{buffName: "stdout", name: w.Name}
	cmd.Stdout = &commandLogger{buffName: "stderr", name: w.Name}
	// Use setpgid to create a process group, so we can send the terminate signal to the
	// processes and all of its children. This only works on unix systems.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Send the terminate signal to the process group by passing the negative pid.
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if err != nil {
			slog.Warn("command: failed to send SIGTERM", "command", w, "error", err)
		}
		return err
	}
	ready <- struct{}{}
	err := cmd.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
