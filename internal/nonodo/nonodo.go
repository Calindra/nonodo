// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the nonodo run function.
// This is separate from the main package to facilitate testing.
package nonodo

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/synchronizer"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/echoapp"
	"github.com/calindra/nonodo/internal/health"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader"
	"github.com/calindra/nonodo/internal/rollup"
	"github.com/calindra/nonodo/internal/salsa"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/calindra/nonodo/internal/sequencers/espresso"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const (
	DefaultHttpPort    = 8080
	DefaultRollupsPort = 5004
	DefaultNamespace   = 10008
)

// Options to nonodo.
type NonodoOpts struct {
	AnvilAddress       string
	AnvilPort          int
	AnvilVerbose       bool
	HttpAddress        string
	HttpPort           int
	HttpRollupsPort    int
	InputBoxAddress    string
	InputBoxBlock      uint64
	ApplicationAddress string
	// If RpcUrl is set, connect to it instead of anvil.
	RpcUrl      string
	EspressoUrl string
	// If set, start echo dapp.
	EnableEcho bool
	// If set, disables devnet.
	DisableDevnet bool
	// If set, disables advances.
	DisableAdvance bool
	// If set, start application.
	ApplicationArgs     []string
	HLGraphQL           bool
	SqliteFile          string
	FromBlock           uint64
	DbImplementation    string
	NodeVersion         string
	LoadTestMode        bool
	Sequencer           string
	Namespace           uint64
	TimeoutInspect      time.Duration
	TimeoutAdvance      time.Duration
	TimeoutWorker       time.Duration
	GraphileUrl         string
	GraphileDisableSync bool
	Salsa               bool
	SalsaUrl            string
	InputPayload        string
	AvailFromBlock      uint64
}

// Create the options struct with default values.
func NewNonodoOpts() NonodoOpts {
	var (
		defaultTimeout time.Duration = 10 * time.Second
		graphileUrl                  = os.Getenv("GRAPHILE_URL")
	)
	const defaultGraphileUrl = "http://localhost:5001/graphql"

	if graphileUrl == "" {
		graphileUrl = defaultGraphileUrl
	}

	// Check if the URL is valid
	if _, err := url.Parse(graphileUrl); err != nil {
		graphileUrl = defaultGraphileUrl
	}

	return NonodoOpts{
		AnvilAddress:        devnet.AnvilDefaultAddress,
		AnvilPort:           devnet.AnvilDefaultPort,
		AnvilVerbose:        false,
		HttpAddress:         "127.0.0.1",
		HttpPort:            DefaultHttpPort,
		HttpRollupsPort:     DefaultRollupsPort,
		InputBoxAddress:     devnet.InputBoxAddress,
		InputBoxBlock:       0,
		ApplicationAddress:  devnet.ApplicationAddress,
		RpcUrl:              "",
		EspressoUrl:         "https://query.cappuccino.testnet.espresso.network",
		EnableEcho:          false,
		DisableDevnet:       false,
		DisableAdvance:      false,
		ApplicationArgs:     nil,
		HLGraphQL:           false,
		SqliteFile:          "file:memory1?mode=memory&cache=shared",
		FromBlock:           0,
		DbImplementation:    "sqlite",
		NodeVersion:         "v1",
		Sequencer:           "inputbox",
		LoadTestMode:        false,
		Namespace:           DefaultNamespace,
		TimeoutInspect:      defaultTimeout,
		TimeoutAdvance:      defaultTimeout,
		TimeoutWorker:       supervisor.DefaultSupervisorTimeout,
		GraphileUrl:         graphileUrl,
		GraphileDisableSync: false,
		Salsa:               false,
		SalsaUrl:            "127.0.0.1:5005",
		InputPayload:        "",
		AvailFromBlock:      0,
	}
}

func NewSupervisorHLGraphQL(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	w.Timeout = opts.TimeoutWorker
	var db *sqlx.DB

	if opts.DbImplementation == "postgres" {
		slog.Info("Using PostGres DB ...")
		postgresHost := os.Getenv("POSTGRES_HOST")
		postgresPort := os.Getenv("POSTGRES_PORT")
		postgresDataBase := os.Getenv("POSTGRES_DB")
		postgresUser := os.Getenv("POSTGRES_USER")
		postgresPassword := os.Getenv("POSTGRES_PASSWORD")

		connectionString := fmt.Sprintf("host=%s port=%s user=%s "+
			"dbname=%s password=%s sslmode=disable",
			postgresHost, postgresPort, postgresUser,
			postgresDataBase, postgresPassword)

		db = sqlx.MustConnect("postgres", connectionString)
	} else {
		slog.Info("Using SQLite ...")
		db = sqlx.MustConnect("sqlite3", opts.SqliteFile)
	}

	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()

	var adapter reader.Adapter

	if opts.NodeVersion == "v1" {
		adapter = reader.NewAdapterV1(db, convenienceService)
	} else {
		graphileUrl, err := url.Parse(opts.GraphileUrl)
		if err != nil {
			slog.Error("Error parsing Graphile URL", "error", err)
			panic(err)
		}

		httpClient := container.GetGraphileClient(*graphileUrl, opts.LoadTestMode)
		inputBlobAdapter := reader.InputBlobAdapter{}
		adapter = reader.NewAdapterV2(convenienceService, httpClient, inputBlobAdapter)
	}

	if opts.RpcUrl == "" && !opts.DisableDevnet {
		anvilLocation, err := handleAnvilInstallation()
		if err != nil {
			panic(err)
		}

		w.Workers = append(w.Workers, devnet.AnvilWorker{
			Address:  opts.AnvilAddress,
			Port:     opts.AnvilPort,
			Verbose:  opts.AnvilVerbose,
			AnvilCmd: anvilLocation,
		})
		opts.RpcUrl = fmt.Sprintf("ws://%s:%v", opts.AnvilAddress, opts.AnvilPort)
	}

	if !opts.LoadTestMode && !opts.GraphileDisableSync {
		slog.Debug("Sync initialization")
		var synchronizer supervisor.Worker

		if opts.NodeVersion == "v2" {
			graphileUrl, err := url.Parse(opts.GraphileUrl)
			if err != nil {
				slog.Error("Error parsing Graphile URL", "error", err)
				panic(err)
			}

			synchronizer = container.GetGraphileSynchronizer(*graphileUrl, opts.LoadTestMode)
		} else {
			synchronizer = container.GetGraphQLSynchronizer()
		}

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
	}

	model := model.NewNonodoModel(
		decoder,
		container.GetReportRepository(),
		container.GetInputRepository(),
	)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      opts.TimeoutInspect,
	}))
	inspect.Register(e, model)
	health.Register(e)
	reader.Register(e, model, convenienceService, adapter)
	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%v:%v", opts.HttpAddress, opts.HttpPort),
		Handler: e,
	})

	if opts.Salsa {
		w.Workers = append(w.Workers, salsa.SalsaWorker{
			Address: opts.SalsaUrl,
		})
	}

	cleanSync := synchronizer.NewCleanSynchronizer(container.GetSyncRepository(), nil)
	w.Workers = append(w.Workers, cleanSync)

	slog.Info("Listening", "port", opts.HttpPort)
	return w
}

func handleAnvilInstallation() (string, error) {
	// Create Anvil Worker
	var timeoutAnvil time.Duration = 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeoutAnvil)
	defer cancel()

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("Timeout waiting for anvil")
		}
	}()

	anvilLocation, err := devnet.CheckAnvilAndInstall(ctx)
	return anvilLocation, err
}

// Create the nonodo supervisor.
func NewSupervisor(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	w.Timeout = opts.TimeoutWorker
	db := sqlx.MustConnect("sqlite3", opts.SqliteFile)
	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)
	modelInstance := model.NewNonodoModel(decoder,
		container.GetReportRepository(),
		container.GetInputRepository(),
	)
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      opts.TimeoutInspect,
	}))
	inspect.Register(e, modelInstance)
	reader.Register(e, modelInstance, convenienceService, adapter)
	health.Register(e)

	// Start the "internal" http rollup server
	re := echo.New()
	re.Use(middleware.CORS())
	re.Use(middleware.Recover())
	re.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      opts.TimeoutAdvance,
	}))

	if opts.RpcUrl == "" && !opts.DisableDevnet {
		anvilLocation, err := handleAnvilInstallation()
		if err != nil {
			panic(err)
		}

		w.Workers = append(w.Workers, devnet.AnvilWorker{
			Address:  opts.AnvilAddress,
			Port:     opts.AnvilPort,
			Verbose:  opts.AnvilVerbose,
			AnvilCmd: anvilLocation,
		})
		opts.RpcUrl = fmt.Sprintf("ws://%s:%v", opts.AnvilAddress, opts.AnvilPort)
	}
	var sequencer model.Sequencer = nil
	if !opts.DisableAdvance {
		if opts.Sequencer == "inputbox" {
			sequencer = model.NewInputBoxSequencer(modelInstance)
			w.Workers = append(w.Workers, inputter.InputterWorker{
				Model:              modelInstance,
				Provider:           opts.RpcUrl,
				InputBoxAddress:    common.HexToAddress(opts.InputBoxAddress),
				InputBoxBlock:      opts.InputBoxBlock,
				ApplicationAddress: common.HexToAddress(opts.ApplicationAddress),
			})
		} else if opts.Sequencer == "espresso" {
			sequencer = model.NewEspressoSequencer(modelInstance)
			w.Workers = append(w.Workers, espresso.NewEspressoListener(
				opts.EspressoUrl,
				opts.Namespace,
				modelInstance.GetInputRepository(),
				opts.FromBlock,
				&inputter.InputterWorker{
					Model:              modelInstance,
					Provider:           opts.RpcUrl,
					InputBoxAddress:    common.HexToAddress(opts.InputBoxAddress),
					InputBoxBlock:      0,
					ApplicationAddress: common.HexToAddress(opts.ApplicationAddress),
				},
			))
		} else if opts.Sequencer == "paio" {
			panic("sequencer not supported yet")
		} else {
			panic("sequencer not supported")
		}
	}

	w.Workers = append(w.Workers, avail.NewAvailListener(opts.AvailFromBlock))

	rollup.Register(re, modelInstance, sequencer)
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
			Env: []string{fmt.Sprintf("ROLLUP_HTTP_SERVER_URL=http://%s:%v",
				opts.HttpAddress, opts.HttpRollupsPort)},
		})
	} else if opts.EnableEcho {
		fmt.Println("Starting echo app")
		w.Workers = append(w.Workers, echoapp.EchoAppWorker{
			RollupEndpoint: fmt.Sprintf("http://%s:%v", opts.HttpAddress, opts.HttpRollupsPort),
		})
	}

	if opts.Salsa {
		w.Workers = append(w.Workers, salsa.SalsaWorker{
			Address: opts.SalsaUrl,
		})
	}

	cleanSync := synchronizer.NewCleanSynchronizer(container.GetSyncRepository(), nil)
	w.Workers = append(w.Workers, cleanSync)

	return w
}
