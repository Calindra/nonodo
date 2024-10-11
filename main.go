// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the main function that executes the nonodo command.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/nonodo"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/calindra/nonodo/internal/sequencers/espresso"
	"github.com/carlmjohnson/versioninfo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	MAX_FILE_SIZE     uint64 = 1_440_000 // 1,44 MB
	APP_ADDRESS              = common.HexToAddress(devnet.ApplicationAddress)
	DEFAULT_NAMESPACE        = 10008
)

var startupMessage = `
Http Rollups for development started at http://localhost:ROLLUPS_PORT
GraphQL running at http://localhost:HTTP_PORT/graphql
Inspect running at http://localhost:HTTP_PORT/inspect/
Press Ctrl+C to stop the node
`

var startupMessageWithLambada = `
Http Rollups for development started at http://localhost:ROLLUPS_PORT
GraphQL running at http://localhost:HTTP_PORT/graphql
Inspect running at http://localhost:HTTP_PORT/inspect/
Lambada running at http://SALSA_URL
Press Ctrl+C to stop the node
`

var cmd = &cobra.Command{
	Use:     "nonodo [flags] [-- application [args]...]",
	Short:   "nonodo is a development node for Cartesi Rollups",
	Run:     run,
	Version: versioninfo.Short(),
}

var CompletionCmd = &cobra.Command{
	Use:                   "completion",
	Short:                 "Generate shell completion scripts",
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cobra.CheckErr(cmd.Root().GenBashCompletion(os.Stdout))
		case "zsh":
			cobra.CheckErr(cmd.Root().GenZshCompletion(os.Stdout))
		case "fish":
			cobra.CheckErr(cmd.Root().GenFishCompletion(os.Stdout, true))
		case "powershell":
			cobra.CheckErr(cmd.Root().GenPowerShellCompletion(os.Stdout))
		}
	},
}

var addressBookCmd = &cobra.Command{
	Use:   "address-book",
	Short: "Show address book",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Debug("Read json and print address...")
		devnet.ShowAddresses()
	},
}

// Celestia Network
type CelestiaOpts struct {
	Payload     string
	PayloadPath string
	PayloadUrl  string
	Namespace   string
	Height      uint64
	Start       uint64
	End         uint64
	RpcUrl      string
	chainId     int64
}

// Espresso
type EspressoOpts struct {
	Payload   string
	Namespace int
}

var celestiaCmd = &cobra.Command{
	Use:   "celestia",
	Short: "Handle blob to Celestia",
	Long:  "Submit a blob and check proofs after one hour to Celestia Network",
}

var espressoCmd = &cobra.Command{
	Use:   "espresso",
	Short: "Handles Espresso transactions",
	Long:  "Submit and get a transaction from Espresso using Cappuccino APIs",
}

type AvailOpts struct {
	Payload     string
	ChainId     int
	AppId       int
	Address     string
	MaxGasPrice uint64
}

var availCmd = &cobra.Command{
	Use:   "avail",
	Short: "Handles Avail transactions",
	Long:  "Submit a transaction to Avail",
}

var (
	debug bool
	color bool
	opts  = nonodo.NewNonodoOpts()
)

func markFlagRequired(cmd *cobra.Command, flagNames ...string) {
	for _, flagName := range flagNames {
		err := cmd.MarkFlagRequired(flagName)
		cobra.CheckErr(err)
	}
}

func ArrBytesAttr(key string, v []byte) slog.Attr {
	var str string
	for _, b := range v {
		s := fmt.Sprintf("%02x", b)
		str += s
	}
	return slog.String(key, str)
}

func CheckIfValidSize(size uint64) error {
	if size > MAX_FILE_SIZE {
		return fmt.Errorf("File size is too big %d bytes", size)
	}

	return nil
}

func addCelestiaSubcommands(celestiaCmd *cobra.Command) {
	celestia := &CelestiaOpts{}

	// Send file
	celestiaSendFileUrlCmd := &cobra.Command{
		Use:   "send-file-url",
		Short: "Send a url file to Celestia Network",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Send a url file to Celestia Network")
			ctx := cmd.Context()

			slog.Info("URL", "url", celestia.PayloadUrl)

			// Download file
			content, err := downloadFile(ctx, celestia.PayloadUrl)
			if err != nil {
				return err
			}

			// Check if the file is valid
			err = CheckIfValidSize(uint64(len(content)))
			if err != nil {
				return err
			}

			slog.Info("File content", ArrBytesAttr("hex", content))
			// slog.Info("File content", slog.String("Content", string(content)))

			token, url, err := getTokenFromTia()
			if err != nil {
				return err
			}

			height, start, end, err := dataavailability.SubmitBlob(ctx, url, token, celestia.Namespace, []byte(celestia.Payload))
			if err != nil {
				slog.Error("Submit", "error", err)
				return err
			}

			slog.Info("Blob was included at", "height", height, "start", start, "end", end)

			return nil
		},
	}
	celestiaSendFileUrlCmd.Flags().StringVar(&celestia.PayloadUrl, "url", "", "File to send to Celestia Network")
	celestiaSendFileUrlCmd.Flags().StringVar(&celestia.Namespace, "namespace", "", "Namespace of the payload")
	markFlagRequired(celestiaSendFileUrlCmd, "url", "namespace")

	celestiaSendFileCmd := &cobra.Command{
		Use:   "send-file",
		Short: "Send a file to Celestia Network",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Send a file to Celestia Network")

			ctx := cmd.Context()

			content, err := readFile(ctx, celestia.PayloadPath)
			if err != nil {
				return err
			}

			// Check if the file is valid
			err = CheckIfValidSize(uint64(len(content)))
			if err != nil {
				return err
			}

			slog.Info("File content", ArrBytesAttr("hex", content))

			token, url, err := getTokenFromTia()
			if err != nil {
				return err
			}

			height, start, end, err := dataavailability.SubmitBlob(ctx, url, token, celestia.Namespace, []byte(celestia.Payload))
			if err != nil {
				slog.Error("Submit", "error", err)
				return err
			}

			slog.Info("Blob was included at", "height", height, "start", start, "end", end)

			return nil
		},
	}
	celestiaSendFileCmd.Flags().StringVar(&celestia.PayloadPath, "file", "", "File to send to Celestia Network")
	celestiaSendFileCmd.Flags().StringVar(&celestia.Namespace, "namespace", "", "Namespace of the payload")
	markFlagRequired(celestiaSendFileCmd, "file", "namespace")
	cobra.CheckErr(celestiaSendFileCmd.MarkFlagFilename("file"))

	// Send
	celestiaSendCmd := &cobra.Command{
		Use:   "send",
		Short: "Send a payload to Celestia Network",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Send a payload to Celestia Network")

			ctx := cmd.Context()

			token, url, err := getTokenFromTia()
			if err != nil {
				return err
			}

			height, start, end, err := dataavailability.SubmitBlob(ctx, url, token, celestia.Namespace, []byte(celestia.Payload))
			if err != nil {
				slog.Error("Submit", "error", err)
				return err
			}

			slog.Info("Blob was included at", "height", height, "start", start, "end", end)

			return nil
		},
	}
	celestiaSendCmd.Flags().StringVar(&celestia.Payload, "payload", "", "Payload to send to Celestia Network")
	celestiaSendCmd.Flags().StringVar(&celestia.Namespace, "namespace", "", "Namespace of the payload")
	markFlagRequired(celestiaSendCmd, "payload", "namespace")

	// Check proof
	celestiaCheckProofCmd := &cobra.Command{
		Use:   "check-proof",
		Short: "Check proof of a payload sent to Celestia Network",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Check proof of a payload sent to Celestia Network")

			ctx := cmd.Context()

			shareProof, dataBlock, err := dataavailability.GetShareProof(
				ctx, celestia.Height, celestia.Start, celestia.End,
			)
			if err != nil {
				return err
			}

			slog.Info("Share Proof", "proof", shareProof, "dataBlock", dataBlock)

			return nil
		},
	}
	celestiaCheckProofCmd.Flags().Uint64Var(&celestia.Height, "height", 0, "Height of the block")
	celestiaCheckProofCmd.Flags().Uint64Var(&celestia.Start, "start", 0, "Start of the proof")
	celestiaCheckProofCmd.Flags().Uint64Var(&celestia.End, "end", 0, "End of the proof")
	markFlagRequired(celestiaCheckProofCmd, "height", "start", "end")

	// Send to relay
	celestiaRelaySend := &cobra.Command{
		Use:   "relay-send",
		Short: "Send a payload to Celestia Relay",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Send a payload to Celestia Relay")

			ctx := cmd.Context()
			err := dataavailability.CallCelestiaRelay(ctx, celestia.Height, celestia.Start, celestia.End, APP_ADDRESS, []byte{}, celestia.RpcUrl, celestia.chainId)
			if err != nil {
				return err
			}

			slog.Info("Payload sent to Celestia Relay")

			return nil
		},
	}
	const goTestnetChainId = 31337
	celestiaRelaySend.Flags().Uint64Var(&celestia.Height, "height", 0, "Height of the block")
	celestiaRelaySend.Flags().Uint64Var(&celestia.Start, "start", 0, "Start of the proof")
	celestiaRelaySend.Flags().Uint64Var(&celestia.End, "end", 0, "End of the proof")
	celestiaRelaySend.Flags().Int64Var(&celestia.chainId, "chain-id", goTestnetChainId, "Chain ID of the network")
	celestiaRelaySend.Flags().StringVar(&celestia.RpcUrl, "rpc-url", "http://localhost:8545",
		"If set, celestia command connects to this url instead of setting up Anvil")
	markFlagRequired(celestiaRelaySend, "height", "start", "end")

	celestiaCmd.AddCommand(celestiaSendCmd, celestiaCheckProofCmd, celestiaRelaySend, celestiaSendFileCmd, celestiaSendFileUrlCmd)
}

func downloadFile(ctx context.Context, url string) ([]byte, error) {
	slog.Info("Download file", "url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error("Create request", "error", err)
		return nil, err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Get file", "error", err)
		return nil, err
	}

	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Read file", "error", err)
		return nil, err
	}
	return content, nil
}

func addEspressoSubcommands(espressoCmd *cobra.Command) {
	espressoOpts := &EspressoOpts{}
	// Send
	espressoSendCmd := &cobra.Command{
		Use:   "send",
		Short: "Send a payload to Espresso",
		RunE: func(cmd *cobra.Command, args []string) error {
			espressoClient := espresso.EspressoClient{
				EspressoUrl: opts.EspressoUrl,
				GraphQLUrl:  fmt.Sprintf("http://%s:%d", opts.HttpAddress, opts.HttpPort),
			}
			_, err := espressoClient.SendInput(espressoOpts.Payload, espressoOpts.Namespace)
			if err != nil {
				panic(err)
			}
			return nil
		},
	}
	espressoSendCmd.Flags().StringVar(&espressoOpts.Payload, "payload", "", "Payload to send to Espresso")
	espressoSendCmd.Flags().IntVar(&espressoOpts.Namespace, "namespace", DEFAULT_NAMESPACE, "Namespace of the payload")
	markFlagRequired(espressoSendCmd, "payload")
	espressoCmd.AddCommand(espressoSendCmd)

}

func addAvailSubcommands(availCmd *cobra.Command) {

	availOpts := &AvailOpts{}

	availSendCmd := &cobra.Command{
		Use:   "send",
		Short: "Send a payload to Avail",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			availClient, err := avail.NewAvailClient(
				fmt.Sprintf("http://%s:%d", opts.HttpAddress, opts.HttpPort),
				availOpts.ChainId,
				availOpts.AppId,
			)
			if err != nil {
				panic(err)
			}
			_, err = availClient.Submit712(ctx, availOpts.Payload, availOpts.Address, availOpts.MaxGasPrice)
			if err != nil {
				panic(err)
			}
			return nil

		},
	}
	availSendCmd.Flags().StringVar(&availOpts.Payload, "payload", "", "Payload to send to Avail")
	availSendCmd.Flags().IntVar(&availOpts.ChainId, "chainId", avail.DEFAULT_CHAINID_HARDHAT, "ChainId used signing EIP-712 messages")
	availSendCmd.Flags().IntVar(&availOpts.AppId, "appId", avail.DEFAULT_APP_ID, "Avail AppId")
	defaultMaxGasPrice := 10
	availSendCmd.Flags().StringVar(&availOpts.Address, "address", devnet.ApplicationAddress, "Address of the dapp")
	availSendCmd.Flags().Uint64Var(&availOpts.MaxGasPrice, "max-gas-price", uint64(defaultMaxGasPrice), "Max gas price")
	markFlagRequired(availSendCmd, "payload")
	availCmd.AddCommand(availSendCmd)
}

func readFile(_ context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Open file", "error", err)
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		slog.Error("Stat file", "error", err)
		return nil, err
	}
	size := stat.Size()
	content := make([]byte, size)
	_, err = file.Read(content)
	if err != nil {
		slog.Error("Read file", "error", err)
		return nil, err
	}
	return content, nil
}

func getTokenFromTia() (tiatoken string, tiaurl string, missingError error) {
	token := os.Getenv("TIA_AUTH_TOKEN")
	url := os.Getenv("TIA_URL")

	if token == "" || url == "" {
		slog.Error("Missing environment variables", "token", token, "url", url)
		return "", "", fmt.Errorf("missing environment variables")
	}
	return token, url, nil
}

func init() {
	// anvil-*
	cmd.Flags().StringVar(&opts.AnvilAddress, "anvil-address", opts.AnvilAddress,
		"HTTP address used by Anvil")
	cmd.Flags().IntVar(&opts.AnvilPort, "anvil-port", opts.AnvilPort,
		"HTTP port used by Anvil")
	cmd.Flags().BoolVar(&opts.AnvilVerbose, "anvil-verbose", opts.AnvilVerbose,
		"If set, prints Anvil's output")

	// contracts-*
	cmd.Flags().StringVar(&opts.ApplicationAddress, "contracts-application-address",
		opts.ApplicationAddress, "Application contract address")
	cmd.Flags().StringVar(&opts.InputBoxAddress, "contracts-input-box-address",
		opts.InputBoxAddress, "InputBox contract address")
	cmd.Flags().Uint64Var(&opts.InputBoxBlock, "contracts-input-box-block",
		opts.InputBoxBlock, "InputBox deployment block number")

	// enable-*
	cmd.Flags().BoolVarP(&debug, "enable-debug", "d", false, "If set, enable debug output")
	cmd.Flags().BoolVar(&color, "enable-color", true, "If set, enables logs color")
	cmd.Flags().BoolVar(&opts.EnableEcho, "enable-echo", opts.EnableEcho,
		"If set, nonodo starts a built-in echo application")

	cmd.Flags().StringVar(&opts.Sequencer, "sequencer", opts.Sequencer,
		"Set the sequencer (inputbox[default] or espresso)")
	cmd.Flags().Uint64Var(&opts.Namespace, "namespace", opts.Namespace,
		"Set the namespace for espresso")
	cmd.Flags().DurationVar(&opts.TimeoutWorker, "timeout-worker", opts.TimeoutWorker, "Timeout for workers. Example: nonodo --timeout-worker 30s")
	cmd.Flags().DurationVar(&opts.TimeoutInspect, "sm-deadline-inspect-state", opts.TimeoutInspect, "Timeout for inspect requests. Example: nonodo --sm-deadline-inspect-state 30s")
	cmd.Flags().DurationVar(&opts.TimeoutAdvance, "sm-deadline-advance-state", opts.TimeoutAdvance, "Timeout for advance requests. Example: nonodo --sm-deadline-advance-state 30s")

	// disable-*
	cmd.Flags().BoolVar(&opts.DisableDevnet, "disable-devnet", opts.DisableDevnet,
		"If set, nonodo won't start a local devnet")
	cmd.Flags().BoolVar(&opts.DisableAdvance, "disable-advance", opts.DisableAdvance,
		"If set, nonodo won't start the inputter to get inputs from the local chain")

	// http-*
	cmd.Flags().StringVar(&opts.HttpAddress, "http-address", opts.HttpAddress,
		"HTTP address used by nonodo to serve its APIs")
	cmd.Flags().IntVar(&opts.HttpPort, "http-port", opts.HttpPort,
		"HTTP port used by nonodo to serve its external APIs")
	cmd.Flags().IntVar(&opts.HttpRollupsPort, "http-rollups-port", opts.HttpRollupsPort,
		"HTTP port used by nonodo to serve its internal APIs")

	// rpc-url
	cmd.Flags().StringVar(&opts.RpcUrl, "rpc-url", opts.RpcUrl,
		"If set, nonodo connects to this url instead of setting up Anvil")

	// convenience experimental implementation
	cmd.Flags().BoolVar(&opts.HLGraphQL, "high-level-graphql", opts.HLGraphQL,
		"If set, enables the convenience layer experiment")

	// database file
	cmd.Flags().StringVar(&opts.SqliteFile, "sqlite-file", opts.SqliteFile,
		"The sqlite file to load the state")

	cmd.Flags().Uint64Var(&opts.FromBlock, "from-block", opts.FromBlock,
		"The beginning of the queried range for events")

	cmd.Flags().StringVar(&opts.DbImplementation, "db-implementation", opts.DbImplementation,
		"DB to use. PostgreSQL or SQLite")

	cmd.Flags().StringVar(&opts.NodeVersion, "node-version", opts.NodeVersion,
		"Node version to emulate")

	cmd.Flags().BoolVar(&opts.LoadTestMode, "load-test-mode", opts.LoadTestMode,
		"If set, enables load test mode")

	cmd.Flags().BoolVar(&opts.GraphileDisableSync, "graphile-disable-sync", opts.GraphileDisableSync,
		"If set, disable graphile synchronization")

	cmd.Flags().StringVar(&opts.GraphileUrl, "graphile-url", opts.GraphileUrl, "URL used to connect to Graphile")

	cmd.Flags().BoolVar(&opts.Salsa, "salsa", opts.Salsa, "If set, starts salsa")

	cmd.Flags().StringVar(&opts.SalsaUrl, "salsa-url", opts.SalsaUrl, "Url used to start Salsa")
	cmd.Flags().BoolVar(&opts.AvailEnabled, "avail-enabled", opts.AvailEnabled, "If set, enables Avail with Paio's sequencer")
	cmd.Flags().Uint64Var(&opts.AvailFromBlock, "avail-from-block", opts.AvailFromBlock, "The beginning of the queried range for events")

	cmd.Flags().StringVar(&opts.PaioServerUrl, "paio-server-url", opts.PaioServerUrl, "The Paio's server url")
}

func run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	startTime := time.Now()

	// setup log
	logOpts := new(tint.Options)
	if debug {
		logOpts.Level = slog.LevelDebug
	}
	logOpts.AddSource = debug
	logOpts.NoColor = !color || !isatty.IsTerminal(os.Stdout.Fd())
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// check args
	checkEthAddress(cmd, "address-input-box")
	checkEthAddress(cmd, "address-application")
	if opts.AnvilPort == 0 {
		exitf("--anvil-port cannot be 0")
	}
	if !cmd.Flags().Changed("sequencer") && cmd.Flags().Changed("rpc-url") && !cmd.Flags().Changed("contracts-input-box-block") {
		exitf("must set --contracts-input-box-block when setting --rpc-url")
	}
	if opts.EnableEcho && len(args) > 0 {
		exitf("can't use built-in echo with custom application")
	}
	opts.ApplicationArgs = args

	// handle signals with notify context
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var startMessage string

	if opts.Salsa {
		startMessage = startupMessageWithLambada
	} else {
		startMessage = startupMessage
	}

	// start nonodo
	ready := make(chan struct{}, 1)
	go func() {
		select {
		case <-ready:
			msg := strings.Replace(
				startMessage,
				"HTTP_PORT",
				fmt.Sprint(opts.HttpPort), -1)
			msg = strings.Replace(
				msg,
				"SALSA_URL",
				fmt.Sprint(opts.SalsaUrl), -1)
			msg = strings.Replace(
				msg,
				"ROLLUPS_PORT",
				fmt.Sprint(opts.HttpRollupsPort), -1)
			fmt.Println(msg)
			slog.Info("nonodo: ready", "after", time.Since(startTime))
		case <-ctx.Done():
		}
	}()
	LoadEnv()
	if opts.HLGraphQL {
		err := nonodo.NewSupervisorHLGraphQL(opts).Start(ctx, ready)
		cobra.CheckErr(err)
	} else {
		opts.AutoCount = true // not check the Idempotency
		err := nonodo.NewSupervisor(opts).Start(ctx, ready)
		cobra.CheckErr(err)
	}
}

//go:embed .env
var envBuilded string

// LoadEnv from embedded .env file
func LoadEnv() {
	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}

	parse, err := godotenv.Unmarshal(envBuilded)
	cobra.CheckErr(err)

	for k, v := range parse {
		if !currentEnv[k] {
			slog.Debug("env: setting env", "key", k, "value", v)
			err := os.Setenv(k, v)
			cobra.CheckErr(err)
		} else {
			slog.Debug("env: skipping env", "key", k)
		}
	}

	slog.Debug("env: loaded")
}

func main() {
	addCelestiaSubcommands(celestiaCmd)
	addEspressoSubcommands(espressoCmd)
	addAvailSubcommands(availCmd)
	cmd.AddCommand(addressBookCmd, celestiaCmd, CompletionCmd, espressoCmd, availCmd)
	cobra.CheckErr(cmd.Execute())
}

func exitf(format string, args ...any) {
	err := fmt.Sprintf(format, args...)
	slog.Error("configuration error", "error", err)
	os.Exit(1)
}

func checkEthAddress(cmd *cobra.Command, varName string) {
	if cmd.Flags().Changed(varName) {
		value, err := cmd.Flags().GetString(varName)
		cobra.CheckErr(err)
		bytes, err := hexutil.Decode(value)
		if err != nil {
			exitf("invalid address for --%v: %v", varName, err)
		}
		if len(bytes) != common.AddressLength {
			exitf("invalid address for --%v: wrong length", varName)
		}
	}
}
