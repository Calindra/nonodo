// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This program gets the devnet state from the devnet Docker image.
// To do that, it creates a container from the image, copies the state file, and deletes the
// container.
package main

import (
	"fmt"
	"os/exec"
	"strings"
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
	// you can see the tags on
	// https://hub.docker.com/r/sunodo/devnet/tags
	// update me when the image is updated
	run("docker", "create", "--name", "temp-devnet", "sunodo/devnet:1.6.0")
	defer run("docker", "rm", "temp-devnet")
	// run("docker", "cp", "temp-devnet:/usr/share/sunodo/anvil_state.json", ".")
	// run("docker", "cp", "temp-devnet:/usr/share/sunodo/localhost.json", ".")
}
