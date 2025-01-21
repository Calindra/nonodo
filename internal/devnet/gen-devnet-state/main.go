// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This program gets the devnet state from the devnet Docker image.
// To do that, it creates a container from the image, copies the state file, and deletes the
// container.
package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
)

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("'%v %v' failed with %v: %v",
			name, strings.Join(args, " "), err, string(output)))
	}
}

func main() {
	commons.ConfigureLog(slog.LevelDebug)
	// you can see the tags on
	// https://github.com/cartesi/cli/pkgs/container/sdk
	// update me when the image is updated
	slog.Info("Creating temporary container")

	// like __filename
	// _, xdir, _, ok := runtime.Caller(0)
	// if !ok {
	// 	panic("no found dirname")
	// }

	// Custom Docker anvil state
	// dockerfile := filepath.Join(filepath.Dir(xdir), "Dockerfile.sdk")
	// run("docker", "build", "-t", "temp-devnet:devrel", ".", "-f", dockerfile)
	// run("docker", "create", "--name", "temp-devnet", "temp-devnet:devrel")

	// Production ready anvil state
	run("docker", "create", "--name", "temp-devnet", "ghcr.io/cartesi/sdk:0.11.0")

	slog.Info("Copying the state file")
	defer func() {
		run("docker", "rm", "temp-devnet")
		// run("docker", "rmi", "temp-devnet:devrel")
		slog.Info("Finished copying the state file")
	}()
	// run("docker", "cp", "temp-devnet:/usr/share/cartesi/anvil_state.json", ".")
	// run("docker", "cp", "temp-devnet:/usr/share/cartesi/localhost.json", ".")
}
