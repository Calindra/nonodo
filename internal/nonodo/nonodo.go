// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the nonodo run function.
// This is separate from the main package to facilitate testing.
package nonodo

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/echoapp"
	"github.com/calindra/nonodo/internal/execlistener"
	"github.com/calindra/nonodo/internal/inputter"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader"
	"github.com/calindra/nonodo/internal/rollup"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const DefaultHttpPort = 8080
const DefaultRollupsPort = 5004
const HttpTimeout = 10 * time.Second

// Options to nonodo.
type NonodoOpts struct {
	AnvilAddress string
	AnvilPort    int
	AnvilVerbose bool

	HttpAddress     string
	HttpPort        int
	HttpRollupsPort int

	InputBoxAddress    string
	InputBoxBlock      uint64
	ApplicationAddress string

	// If RpcUrl is set, connect to it instead of anvil.
	RpcUrl string

	// If set, start echo dapp.
	EnableEcho bool

	// If set, disables devnet.
	DisableDevnet bool

	// If set, disables advances.
	DisableAdvance bool

	// If set, start application.
	ApplicationArgs []string

	OnlyExecListener bool
}

// Create the options struct with default values.
func NewNonodoOpts() NonodoOpts {
	return NonodoOpts{
		AnvilAddress:       devnet.AnvilDefaultAddress,
		AnvilPort:          devnet.AnvilDefaultPort,
		AnvilVerbose:       false,
		HttpAddress:        "127.0.0.1",
		HttpPort:           DefaultHttpPort,
		HttpRollupsPort:    DefaultRollupsPort,
		InputBoxAddress:    devnet.InputBoxAddress,
		InputBoxBlock:      0,
		ApplicationAddress: devnet.ApplicationAddress,
		RpcUrl:             "",
		EnableEcho:         false,
		DisableDevnet:      false,
		DisableAdvance:     false,
		ApplicationArgs:    nil,
		OnlyExecListener:   true,
	}
}

// Create the nonodo supervisor.
func NewSupervisor(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker

	model := model.NewNonodoModel()
	if opts.OnlyExecListener {
		opts.RpcUrl = fmt.Sprintf("ws://%s:%v", opts.AnvilAddress, opts.AnvilPort)
		// opts.HttpPort = 8181
		w.Workers = append(w.Workers, execlistener.ExecListener{
			Model:              model,
			Provider:           opts.RpcUrl,
			ApplicationAddress: common.HexToAddress(opts.ApplicationAddress),
		})
		e := echo.New()
		e.Use(middleware.CORS())
		e.Use(middleware.Recover())
		e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
			ErrorMessage: "Request timed out",
			Timeout:      HttpTimeout,
		}))
		inspect.Register(e, model)
		reader.Register(e, model)
		w.Workers = append(w.Workers, supervisor.HttpWorker{
			Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
			Handler: e,
		})
		slog.Info("Listening", "port", opts.HttpPort)
		return w
	}

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      HttpTimeout,
	}))
	inspect.Register(e, model)
	reader.Register(e, model)

	// Start the "internal" http rollup server
	re := echo.New()
	re.Use(middleware.CORS())
	re.Use(middleware.Recover())
	re.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      HttpTimeout,
	}))
	rollup.Register(re, model)

	if opts.RpcUrl == "" && !opts.DisableDevnet {
		w.Workers = append(w.Workers, devnet.AnvilWorker{
			Address: opts.AnvilAddress,
			Port:    opts.AnvilPort,
			Verbose: opts.AnvilVerbose,
		})
		opts.RpcUrl = fmt.Sprintf("ws://%s:%v", opts.AnvilAddress, opts.AnvilPort)
	}
	if !opts.DisableAdvance {
		w.Workers = append(w.Workers, inputter.InputterWorker{
			Model:              model,
			Provider:           opts.RpcUrl,
			InputBoxAddress:    common.HexToAddress(opts.InputBoxAddress),
			InputBoxBlock:      opts.InputBoxBlock,
			ApplicationAddress: common.HexToAddress(opts.ApplicationAddress),
		})
	}
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, DefaultRollupsPort),
		Handler: re,
	})
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
		Handler: e,
	})
	if len(opts.ApplicationArgs) > 0 {
		fmt.Println("Starting app with supervisor")
		w.Workers = append(w.Workers, supervisor.CommandWorker{
			Name:    "app",
			Command: opts.ApplicationArgs[0],
			Args:    opts.ApplicationArgs[1:],
			Env: []string{fmt.Sprintf("ROLLUP_HTTP_SERVER_URL=http://%s:%v/rollup",
				opts.HttpAddress, opts.HttpPort)},
		})
	} else if opts.EnableEcho {
		fmt.Println("Starting echo app")
		w.Workers = append(w.Workers, echoapp.EchoAppWorker{
			RollupEndpoint: fmt.Sprintf("http://127.0.0.1:%v", opts.HttpRollupsPort),
		})
	}
	return w
}
