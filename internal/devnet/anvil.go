// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package devnet

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/supervisor"
)

// Default port for the Ethereum node.
const (
	AnvilDefaultAddress = "127.0.0.1"
	AnvilDefaultPort    = 8545
)

// Generate the devnet state and embed it in the Go binary.
//
//go:generate go run ./gen-devnet-state
//go:embed anvil_state.json
var devnetState []byte

//go:embed localhost.json
var localhost []byte

const StateFileName = "anvil_state.json"

const anvilCommand = "anvil"

// Start the anvil process in the host machine.
type AnvilWorker struct {
	Address            string
	Port               int
	Verbose            bool
	AnvilCmd           string
	AnvilBlockTime     time.Duration
	AnvilStateFileName *string
}

// Define a struct to represent the structure of your JSON data
type ContractInfo struct {
	Contracts map[string]struct {
		Address string `json:"address"`
	} `json:"contracts"`
}

func (w AnvilWorker) String() string {
	return anvilCommand
}

// Try to install anvil
func InstallAnvil(ctx context.Context) (string, error) {
	anvilClient := commons.NewAnvilRelease()
	anvilExec, err := commons.HandleReleaseExecution(ctx, anvilClient)
	if err != nil {
		return "", fmt.Errorf("anvil: %w", err)
	}
	return anvilExec, nil
}

func GetAnvilVersion(anvilExec string) (string, error) {
	output, err := exec.Command(anvilExec, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("anvil: failed to run anvil: %w", err)
	}
	anvilVersion := strings.TrimSpace(string(output))
	return anvilVersion, nil
}

// For while, we dont check if anvil is already installed.
// We always install anvil.
func CheckAnvilAndInstall(ctx context.Context) (string, error) {
	location, err := InstallAnvil(ctx)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to install %w", err)
	}
	anvilVersion, err := GetAnvilVersion(location)
	if err != nil {
		return "", err
	}
	slog.Debug("anvil: installed anvil", "location", location, "version", anvilVersion)
	return location, nil
}

func GetContractInfo() *ContractInfo {
	var contracts ContractInfo
	if err := json.Unmarshal(localhost, &contracts); err != nil {
		slog.Warn("anvil: failed to unmarshal localhost.json", "error", err)
		return nil
	}
	return &contracts
}

func ShowAddresses() {
	contracts := GetContractInfo()
	var names []string
	for name := range contracts.Contracts {
		names = append(names, name)
	}
	names = append(names, ApplicationContractName)
	contracts.Contracts[ApplicationContractName] = struct {
		Address string "json:\"address\""
	}{
		Address: ApplicationAddress,
	}
	sort.Strings(names)
	space := 28
	addressSpace := 42
	fmt.Printf("%-28s %s\n", "Contract", "Address")
	fmt.Printf("%-28s %s\n", strings.Repeat("─", space), strings.Repeat("─", addressSpace))
	for _, name := range names {
		if contract, ok := contracts.Contracts[name]; ok {
			fmt.Printf("%-28s %s\n", name, contract.Address)
		}
	}
}

func (w AnvilWorker) loadState() ([]byte, error) {
	if w.AnvilStateFileName == nil || *w.AnvilStateFileName == StateFileName {
		slog.Debug("anvil: using embedded state file")
		return devnetState, nil
	}
	slog.Debug("anvil: using external state file", "file", *w.AnvilStateFileName)
	content, err := os.ReadFile(*w.AnvilStateFileName)
	if err != nil {
		return nil, fmt.Errorf("anvil: failed to read state file: %w", err)
	}
	return content, nil
}

func (w AnvilWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	content, err := w.loadState()
	if err != nil {
		return err
	}
	dir, err := makeStateTemp(content)
	if err != nil {
		return err
	}
	defer removeTemp(dir)
	slog.Debug("anvil: created temp dir with state file", "dir", dir)

	if w.AnvilCmd == "" {
		w.AnvilCmd = anvilCommand
	}

	var seconds uint64 = uint64(w.AnvilBlockTime.Seconds())

	var server supervisor.ServerWorker
	server.Name = anvilCommand
	server.Command = w.AnvilCmd
	server.Port = w.Port
	server.Args = append(server.Args, "--host", fmt.Sprint(w.Address))
	server.Args = append(server.Args, "--port", fmt.Sprint(w.Port))
	server.Args = append(server.Args, "--load-state", filepath.Join(dir, StateFileName))
	if seconds > 0 {
		server.Args = append(server.Args, "--block-time", fmt.Sprint(seconds))
	}
	// server.Args = append(server.Args, "--tracing")
	if !w.Verbose {
		server.Args = append(server.Args, "--silent")
	}
	return server.Start(ctx, ready)
}

// Create a temporary directory with the state file in it.
// The directory should be removed by the callee.
func makeStateTemp(content []byte) (string, error) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("anvil: failed to create temp dir: %w", err)
	}
	stateFile := filepath.Join(tempDir, StateFileName)
	const permissions = 0644
	err = os.WriteFile(stateFile, content, permissions)
	if err != nil {
		return "", fmt.Errorf("anvil: failed to write state file: %w", err)
	}
	return tempDir, nil
}

// Delete the temporary directory.
func removeTemp(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		slog.Warn("anvil: failed to remove temp file", "error", err)
	}
}
