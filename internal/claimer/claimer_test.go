package claimer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

type ClaimerSuite struct {
	suite.Suite
	ctx           context.Context
	workerCtx     context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
}

func TestCreateProofsSuite(t *testing.T) {
	suite.Run(t, new(ClaimerSuite))
}

func (s *ClaimerSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	var w supervisor.SupervisorWorker
	w.Name = "WorkerClaimerSuite"
	const testTimeout = 5 * time.Second
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:  devnet.AnvilDefaultAddress,
		Port:     devnet.AnvilDefaultPort,
		Verbose:  true,
		AnvilCmd: "anvil",
	})

	s.rpcUrl = fmt.Sprintf("ws://%s:%v", devnet.AnvilDefaultAddress, devnet.AnvilDefaultPort)
	ready := make(chan struct{})
	go func() {
		s.workerResult <- w.Start(s.workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("nonodo ready")
	}
}

func (s *ClaimerSuite) TearDownTest() {
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	s.T().Log("teardown ok.")
}

func (s *ClaimerSuite) TestMakeTheClaim() {
	// dbFactory := commons.NewDbFactory()
	// db := dbFactory.CreateDb("claim.sqlite3")
	// container := convenience.NewContainer(*db, false)
	ctx := context.Background()
	ethClient, err := ethclient.DialContext(ctx, s.rpcUrl)
	s.Require().NoError(err)

	claimer := NewClaimer(ethClient)

	consensusAddress, err := claimer.CreateConsensusTypeAuthority(ctx)
	s.Require().NoError(err)
	slog.Debug("NewAuthority0", "authorityAddress", consensusAddress)

	appContract, err := claimer.Deploy(ctx, *consensusAddress)
	s.Require().NoError(err)
	slog.Debug("Deploy", "appContract", appContract.Hex())

	lastProcessedBlockNumber := new(big.Int).SetUint64(10)
	txOpts, err := devnet.DefaultTxOpts(ctx, ethClient)
	s.Require().NoError(err)

	// nolint
	voucherPayloadStr := "237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000005deadbeef14000000000000000000000000000000000000000000000000000000"
	voucherOutput0 := NewUnifiedOutput(voucherPayloadStr, 0)
	voucherOutput1 := NewUnifiedOutput(voucherPayloadStr, 1)

	outputs := []*UnifiedOutput{
		voucherOutput0, voucherOutput1,
	}
	claimHash, _, err := claimer.CreateClaimRootHash(outputs)
	s.Require().NoError(err)

	err = claimer.MakeTheClaim(
		ctx, consensusAddress, appContract, claimHash, lastProcessedBlockNumber,
		txOpts,
	)
	s.Require().NoError(err)

	applicationOnChain, err := contracts.NewApplication(*appContract, ethClient)
	s.Require().NoError(err)

	callOpts := bind.CallOpts{}
	owner, err := applicationOnChain.Owner(&callOpts)
	s.Require().NoError(err)
	slog.Debug("Owner", "owner", owner, "appContract", appContract.Hex())

	s.Require().Equal(63, len(voucherOutput0.proof.OutputHashesSiblings))

	err = applicationOnChain.ValidateOutput(&callOpts, voucherOutput0.payload, voucherOutput0.proof)
	s.Require().NoError(err)

	txOpts, err = devnet.DefaultTxOpts(ctx, ethClient)
	s.Require().NoError(err)

	_, err = applicationOnChain.ExecuteOutput(txOpts, voucherOutput0.payload, voucherOutput0.proof)
	s.Require().NoError(err)
}

func ConvertSiblings(jsonData string) [][32]byte {

	var siblings []string

	// Parse JSON data into the slice
	if err := json.Unmarshal([]byte(jsonData), &siblings); err != nil {
		panic(err)
	}

	// Print each sibling hash
	asBytes := [][32]byte{}
	for _, hexStr := range siblings {
		// Remove the "0x" prefix if present
		if hexStr[:2] == "0x" {
			hexStr = hexStr[2:]
		}

		// Decode the hex string to bytes
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			panic(err)
		}

		// Convert the bytes to a [32]byte array
		var arr [32]byte
		copy(arr[:], bytes)

		// Append to the slice
		asBytes = append(asBytes, arr)
	}
	return asBytes
}
