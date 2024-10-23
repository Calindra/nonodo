package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerOutputUpdateSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerOutputUpdate   *SynchronizerOutputUpdate
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerOutputUpdateSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerOutputUpdateSuite) SetupTest() {
	s.ctx = context.Background()
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "output.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(*db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNodeV2Repository = NewRawRepository(RAW_DB_URL, dbNodeV2)

	s.synchronizerOutputUpdate = NewSynchronizerOutputUpdate(
		s.container.GetVoucherRepository(),
		s.container.GetNoticeRepository(),
		s.rawNodeV2Repository,
		s.container.GetRawOutputRefRepository(),
	)
}

func (s *SynchronizerOutputUpdateSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerOutputUpdateSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerOutputUpdateSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerOutputUpdateSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerOutputUpdateSuite) TestUpdateOutputs() {
	ctx := context.Background()

	s.fillRefData(ctx)

	// check setup
	proofCount := s.countHLProofs(ctx)
	s.Require().Equal(0, proofCount)

	// first call
	err := s.synchronizerOutputUpdate.SyncOutputs(ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerOutputUpdate.SyncOutputs(ctx)
	s.Require().NoError(err)
	second := s.countHLProofs(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerOutputUpdateSuite) countHLProofs(ctx context.Context) int {
	total, err := s.container.GetOutputRepository().CountProofs(ctx)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerOutputUpdateSuite) fillRefData(ctx context.Context) {
	appContract := common.HexToAddress("0x5112cf49f2511ac7b13a032c4c62a48410fc28fb")
	// msgSender := common.HexToAddress(devnet.SenderAddress)
	for i := 0; i < TOTAL_INPUT_TEST*2; i++ {
		// id := strconv.FormatInt(int64(i), 10) // our ID
		outputType := repository.RAW_VOUCHER_TYPE
		if i%2 == 0 {
			outputType = "notice"
		}
		err := s.container.GetRawOutputRefRepository().Create(ctx, repository.RawOutputRef{
			RawID:       uint64(i + 1),
			InputIndex:  uint64(i),
			OutputIndex: uint64(i),
			AppContract: appContract.Hex(),
			Type:        outputType,
			HasProof:    false,
		})
		s.Require().NoError(err)
		if outputType == repository.RAW_VOUCHER_TYPE {
			_, err = s.container.GetVoucherRepository().CreateVoucher(
				ctx, &model.ConvenienceVoucher{
					AppContract: appContract,
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
				},
			)
			s.Require().NoError(err)
		} else {
			_, err = s.container.GetNoticeRepository().Create(
				ctx, &model.ConvenienceNotice{
					AppContract: appContract.Hex(),
					OutputIndex: uint64(i),
					InputIndex:  uint64(i),
				},
			)
			s.Require().NoError(err)
		}
	}
}
