package claimer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/suite"
)

type ClaimerServiceSuite struct {
	suite.Suite
	ctx            context.Context
	workerCtx      context.Context
	timeoutCancel  context.CancelFunc
	workerCancel   context.CancelFunc
	workerResult   chan error
	rpcUrl         string
	claimerService *ClaimerService
	container      *convenience.Container
	claimer        *Claimer
	ethClient      *ethclient.Client
	dbFactory      *commons.DbFactory
}

func TestClaimerServiceSuite(t *testing.T) {
	suite.Run(t, new(ClaimerServiceSuite))
}

func (s *ClaimerServiceSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	var w supervisor.SupervisorWorker
	w.Name = "WorkerClaimerServiceSuite"
	const testTimeout = 15 * time.Second
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:        devnet.AnvilDefaultAddress,
		Port:           devnet.AnvilDefaultPort,
		Verbose:        true,
		AnvilCmd:       "anvil",
		AnvilBlockTime: 1 * time.Second,
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

	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("claim-service.sqlite3")
	s.container = convenience.NewContainer(*db, true)
	ethClient, err := ethclient.DialContext(s.ctx, s.rpcUrl)
	s.Require().NoError(err)
	s.ethClient = ethClient
	s.claimer = NewClaimer(s.ethClient)
	s.claimerService = NewClaimService(
		s.container.GetVoucherRepository(),
		s.container.GetNoticeRepository(),
		s.claimer,
	)
}

func (s *ClaimerServiceSuite) TearDownTest() {
	s.workerCancel()
	defer s.dbFactory.Cleanup()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	s.T().Log("teardown ok.")
}

func (s *ClaimerServiceSuite) TestMakeTheClaimAndValidateOutputOnChain() {
	consensusAddress, err := s.claimer.CreateConsensusTypeAuthority(s.ctx)
	s.Require().NoError(err)
	appContract, err := s.claimer.CreateNewOnChainApp(s.ctx, *consensusAddress)
	s.Require().NoError(err)
	s.fillData(s.ctx, appContract)
	startBlock := 1
	endBlockLt := 10

	err = s.claimerService.CreateProofsAndSendClaim(
		s.ctx,
		*consensusAddress,
		uint64(startBlock),
		uint64(endBlockLt),
	)
	s.Require().NoError(err)
	vouchers, err := s.container.GetVoucherRepository().FindAll(s.ctx)
	s.Require().NoError(err)
	siblings := []string{}
	voucher := vouchers[0]
	err = json.Unmarshal([]byte(voucher.OutputHashesSiblings), &siblings)
	s.Require().NoError(err)
	s.Equal(63, len(siblings))
	s.checkVoucher(voucher)
}

func (s *ClaimerServiceSuite) checkVoucher(voucher model.ConvenienceVoucher) {
	appContract := voucher.AppContract
	applicationOnChain, err := contracts.NewApplication(appContract, s.ethClient)
	s.Require().NoError(err)

	callOpts := bind.CallOpts{}
	owner, err := applicationOnChain.Owner(&callOpts)
	s.Require().NoError(err)
	slog.Debug("Owner", "owner", owner, "appContract", appContract.Hex())
	time.Sleep(time.Second * 3)
	voucherOutput0 := NewUnifiedOutput(voucher.Payload, voucher.OutputIndex)
	arr, err := To32ByteArray(voucher.OutputHashesSiblings)
	s.Require().NoError(err)
	// arr[0][0] = 0
	voucherOutput0.proof.OutputHashesSiblings = arr
	{
		// it should be valid
		err = applicationOnChain.ValidateOutput(&callOpts, voucherOutput0.payload, voucherOutput0.proof)
		s.Require().NoError(err)
	}
	{
		// it should be executed
		txOpts, err := devnet.DefaultTxOpts(s.ctx, s.ethClient)
		s.Require().NoError(err)
		s.Require().Equal(1, int(voucherOutput0.proof.OutputIndex))
		tx, err := applicationOnChain.ExecuteOutput(txOpts, voucherOutput0.payload, voucherOutput0.proof)
		s.Require().NoError(err)
		receipt, err := bind.WaitMined(context.Background(), s.ethClient, tx)
		s.Require().NoError(err)
		s.Equal(uint64(1), receipt.Status)
	}
	{
		// still valid
		err = applicationOnChain.ValidateOutput(&callOpts, voucherOutput0.payload, voucherOutput0.proof)
		s.Require().NoError(err)
	}
	{
		// it should not be executed again
		txOpts, err := devnet.DefaultTxOpts(s.ctx, s.ethClient)
		s.Require().NoError(err)
		tx, err := applicationOnChain.ExecuteOutput(txOpts, voucherOutput0.payload, voucherOutput0.proof)
		s.Require().NoError(err)
		receipt, err := bind.WaitMined(context.Background(), s.ethClient, tx)
		if err != nil {
			s.Require().NoError(err)
		}
		s.Equal(uint64(0), receipt.Status)
	}

}

const TOTAL_INPUT_TEST = 10

// nolint
func (s *ClaimerServiceSuite) fillData(ctx context.Context, appContract *common.Address) {
	blockNumber := 9
	voucherPayloadStr := "0x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000004deadbeef00000000000000000000000000000000000000000000000000000000"
	noticePayloadStr := "0xc258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005deadbeef00000000000000000000000000000000000000000000000000000000"
	// msgSender := common.HexToAddress(devnet.SenderAddress)
	for i := 0; i < TOTAL_INPUT_TEST*2; i++ {
		id := strconv.FormatInt(int64(i), 10) // our ID
		outputType := repository.RAW_VOUCHER_TYPE
		if i%2 == 0 {
			outputType = repository.RAW_NOTICE_TYPE
		}
		_, err := s.container.GetInputRepository().Create(ctx, model.AdvanceInput{
			ID:          id,
			BlockNumber: uint64(blockNumber),
			AppContract: *appContract,
			Index:       i,
		})
		s.Require().NoError(err)

		if outputType == repository.RAW_VOUCHER_TYPE {
			_, err := s.container.GetVoucherRepository().CreateVoucher(
				ctx, &model.ConvenienceVoucher{
					AppContract: *appContract,
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
					Payload:     voucherPayloadStr,
					Value:       "0x0000000000000000000000000000000000000000000000000000000000000000",
				},
			)
			s.Require().NoError(err)
		} else {
			_, err := s.container.GetNoticeRepository().Create(
				ctx, &model.ConvenienceNotice{
					AppContract: appContract.Hex(),
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
					Payload:     noticePayloadStr,
				},
			)
			s.Require().NoError(err)
		}
	}
}

func To32ByteArray(jsonInput string) ([][32]byte, error) {
	var hexStrings []string
	if err := json.Unmarshal([]byte(jsonInput), &hexStrings); err != nil {
		return nil, err
	}
	var result [][32]byte

	for _, hexStr := range hexStrings {
		// Remove "0x" prefix if present
		if len(hexStr) >= 2 && hexStr[:2] == "0x" {
			hexStr = hexStr[2:]
		}

		// Decode hex string to bytes
		decoded, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("error decoding hex string: %v", err)
		}

		// Ensure the byte slice has exactly 32 bytes
		if len(decoded) != 32 {
			return nil, fmt.Errorf("hex string must be 32 bytes long, got %d bytes", len(decoded))
		}

		// Convert to [32]byte and add to result
		var byte32 [32]byte
		copy(byte32[:], decoded)
		result = append(result, byte32)
	}

	return result, nil
}
