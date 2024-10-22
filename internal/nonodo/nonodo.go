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
	"path"
	"time"

	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/synchronizer"
	synchronizernode "github.com/calindra/nonodo/internal/convenience/synchronizer_node"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/echoapp"
	"github.com/calindra/nonodo/internal/health"
	"github.com/calindra/nonodo/internal/inspect"
	"github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/paio"
	"github.com/calindra/nonodo/internal/reader"
	"github.com/calindra/nonodo/internal/rollup"
	"github.com/calindra/nonodo/internal/salsa"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/calindra/nonodo/internal/sequencers/espresso"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
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
	AutoCount          bool
	AnvilAddress       string
	AnvilPort          int
	AnvilVerbose       bool
	AnvilCommand       string
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
	AvailFromBlock      uint64
	AvailEnabled        bool
	PaioServerUrl       string
	DbRawUrl            string
	RawEnabled          bool
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
		AnvilCommand:        "",
		AnvilVerbose:        false,
		HttpAddress:         "127.0.0.1",
		HttpPort:            DefaultHttpPort,
		HttpRollupsPort:     DefaultRollupsPort,
		InputBoxAddress:     devnet.InputBoxAddress,
		InputBoxBlock:       0,
		ApplicationAddress:  devnet.ApplicationAddress,
		RpcUrl:              "",
		EspressoUrl:         "https://query.decaf.testnet.espresso.network",
		EnableEcho:          false,
		DisableDevnet:       false,
		DisableAdvance:      false,
		ApplicationArgs:     nil,
		HLGraphQL:           false,
		SqliteFile:          "",
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
		AvailFromBlock:      0,
		AvailEnabled:        false,
		AutoCount:           false,
		PaioServerUrl:       "https://cartesi-paio-avail-turing.fly.dev",
		DbRawUrl:            "postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable",
		RawEnabled:          false,
	}
}

func NewSupervisorHLGraphQL(opts NonodoOpts) supervisor.SupervisorWorker {
	var w supervisor.SupervisorWorker
	w.Timeout = opts.TimeoutWorker
	db := CreateDBInstance(opts)
	container := convenience.NewContainer(*db, opts.AutoCount)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)
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
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
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

	if opts.RawEnabled {
		dbNodeV2 := sqlx.MustConnect("postgres", opts.DbRawUrl)
		rawRepository := synchronizernode.NewRawRepository(opts.DbRawUrl, dbNodeV2)
		synchronizerUpdate := synchronizernode.NewSynchronizerUpdate(
			container.GetRawInputRepository(),
			rawRepository,
			container.GetInputRepository(),
		)
		synchronizerReport := synchronizernode.NewSynchronizerReport(
			container.GetReportRepository(),
			rawRepository,
		)
		rawSequencer := synchronizernode.NewSynchronizerCreateWorker(
			container.GetInputRepository(),
			container.GetRawInputRepository(),
			opts.DbRawUrl,
			rawRepository,
			&synchronizerUpdate,
			container.GetOutputDecoder(),
			synchronizerReport,
		)
		w.Workers = append(w.Workers, rawSequencer)
	}

	cleanSync := synchronizer.NewCleanSynchronizer(container.GetSyncRepository(), nil)
	w.Workers = append(w.Workers, cleanSync)

	slog.Info("Listening", "port", opts.HttpPort)
	return w
}

func CreateDBInstance(opts NonodoOpts) *sqlx.DB {
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
		db = handleSQLite(opts)
	}
	return db
}

func handleSQLite(opts NonodoOpts) *sqlx.DB {
	slog.Info("Using SQLite ...")
	sqliteFile := opts.SqliteFile
	if sqliteFile == "" {
		sqlitePath, err := os.MkdirTemp("", "nonodo-db-*")
		if err != nil {
			panic(err)
		}
		sqliteFile = path.Join(sqlitePath, "nonodo.sqlite3")
		slog.Debug("SQLite3 file created", "path", sqliteFile)
	}

	return sqlx.MustConnect("sqlite3", sqliteFile)
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
	db := CreateDBInstance(opts)
	container := convenience.NewContainer(*db, opts.AutoCount)
	decoder := container.GetOutputDecoder()
	convenienceService := container.GetConvenienceService()
	adapter := reader.NewAdapterV1(db, convenienceService)
	modelInstance := model.NewNonodoModel(decoder,
		container.GetReportRepository(),
		container.GetInputRepository(),
		container.GetVoucherRepository(),
		container.GetNoticeRepository(),
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
		anvilLocation := opts.AnvilCommand
		if anvilLocation == "" {
			al, err := handleAnvilInstallation()
			if err != nil {
				panic(err)
			}
			anvilLocation = al
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
	inputterWorker := &inputter.InputterWorker{
		Model:              modelInstance,
		Provider:           opts.RpcUrl,
		InputBoxAddress:    common.HexToAddress(opts.InputBoxAddress),
		InputBoxBlock:      0,
		ApplicationAddress: common.HexToAddress(opts.ApplicationAddress),
	}

	if !opts.DisableAdvance && !opts.AvailEnabled {
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
				inputterWorker,
			))
		} else if opts.Sequencer == "paio" {
			panic("sequencer not supported yet")
		} else {
			panic("sequencer not supported")
		}
	}

	paioSequencerBuilder := paio.NewPaioBuilder()
	paioSequencerBuilder.WithInputRepository(container.GetInputRepository())
	paioSequencerBuilder.WithRpcUrl(opts.RpcUrl)

	if opts.AvailEnabled {
		availClient, err := avail.NewAvailClient(
			fmt.Sprintf("http://%s:%d", opts.HttpAddress, opts.HttpPort),
			avail.DEFAULT_CHAINID_HARDHAT,
			avail.DEFAULT_APP_ID,
		)
		if err != nil {
			panic(err)
		}
		paioSequencerBuilder.WithPaioServerUrl(
			opts.PaioServerUrl,
		)
		paioSequencerBuilder.WithAvalClient(availClient)
	}
	if opts.Sequencer == "espresso" {
		paioSequencerBuilder = paioSequencerBuilder.WithEspressoUrl(opts.EspressoUrl)
	}
	paioSequencer := paioSequencerBuilder.Build()
	paio.Register(e, paioSequencer)

	if opts.AvailEnabled {
		// Check if paio decoder is installed
		paioLocation, err := paiodecoder.DownloadPaioDecoderExecutableAsNeeded()
		if err != nil {
			panic(err)
		}
		slog.Debug("AvailEnabled", "paioLocation", paioLocation)
		w.Workers = append(w.Workers, avail.NewAvailListener(
			opts.AvailFromBlock,
			modelInstance.GetInputRepository(),
			inputterWorker,
			opts.FromBlock,
			paioLocation,
			opts.ApplicationAddress,
		))
		sequencer = model.NewInputBoxSequencer(modelInstance)
	}

	if opts.RawEnabled {
		panic("use the --high-level-graphql flag")
	}

	rollup.Register(re, modelInstance, sequencer, common.HexToAddress(opts.ApplicationAddress))
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
