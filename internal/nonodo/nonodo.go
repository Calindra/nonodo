// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the nonodo run function.
// This is separate from the main package to facilitate testing.
package nonodo

import (
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/echoapp"
	"github.com/calindra/nonodo/internal/inputter"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader"
	"github.com/calindra/nonodo/internal/rollup"
	v1 "github.com/calindra/nonodo/internal/rollup/v1"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
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

	ConveniencePoC   bool
	SqliteFile       string
	FromBlock        uint64
	DbImplementation string

	// If set, enables legacy mode.
	LegacyMode bool
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
		ConveniencePoC:     false,
		SqliteFile:         "file:memory1?mode=memory&cache=shared",
		FromBlock:          0,
		DbImplementation:   "sqlite",
		LegacyMode:         true,
	}
}

func NewSupervisorPoC(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker

	var db *sqlx.DB

	if opts.DbImplementation == "postgres" {
		slog.Info("Using PostGres DB ...")
		postGresHost := os.Getenv("POSTGRES_HOST")
		postGresPort := os.Getenv("POSTGRES_PORT")
		postGresDataBase := os.Getenv("POSTGRES_DB")
		postGresUser := os.Getenv("POSTGRES_USER")
		postGresPassword := os.Getenv("POSTGRES_PASSWORD")

		connectionString := fmt.Sprintf("host=%s port=%s user=%s "+
			"dbname=%s password=%s sslmode=disable",
			postGresHost, postGresPort, postGresUser,
			postGresDataBase, postGresPassword)

		db = sqlx.MustConnect("postgres", connectionString)
	} else {
		slog.Info("Using SQLite ...")
		db = sqlx.MustConnect("sqlite3", opts.SqliteFile)
	}

	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)
	synchronizer := container.GetGraphQLSynchronizer()
	model := model.NewNonodoModel(decoder, db)
	w.Workers = append(w.Workers, synchronizer)
	opts.RpcUrl = fmt.Sprintf("ws://%s:%v", opts.AnvilAddress, opts.AnvilPort)
	fromBlock := new(big.Int).SetUint64(opts.FromBlock)
	execVoucherListener := convenience.NewExecListener(
		opts.RpcUrl,
		common.HexToAddress(opts.ApplicationAddress),
		convenienceService,
		fromBlock,
	)
	w.Workers = append(w.Workers, execVoucherListener)
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      HttpTimeout,
	}))
	inspect.Register(e, model)
	reader.Register(e, model, convenienceService, adapter)
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
		Handler: e,
	})
	slog.Info("Listening", "port", opts.HttpPort)
	return w
}

// Create the nonodo supervisor.
func NewSupervisor(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	db := sqlx.MustConnect("sqlite3", opts.SqliteFile)
	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)
	model := model.NewNonodoModel(decoder, db)
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      HttpTimeout,
	}))
	inspect.Register(e, model)
	reader.Register(e, model, convenienceService, adapter)

	// Start the "internal" http rollup server
	re := echo.New()
	re.Use(middleware.CORS())
	re.Use(middleware.Recover())
	re.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      HttpTimeout,
	}))

	if opts.LegacyMode {
		v1.Register(re, model)
	} else {
		rollup.Register(re, model)
	}

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
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpRollupsPort),
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
