package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerOutputCreateSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerOutputCreate   *SynchronizerOutputCreate
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerOutputCreateSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerOutputCreateSuite) SetupTest() {
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

	abi, err := contracts.OutputsMetaData.GetAbi()
	if err != nil {
		s.Require().NoError(err)
	}
	abiDecoder := NewAbiDecoder(abi)
	s.synchronizerOutputCreate = NewSynchronizerOutputCreate(
		s.container.GetVoucherRepository(),
		s.container.GetNoticeRepository(),
		s.rawNodeV2Repository,
		s.container.GetRawOutputRefRepository(),
		abiDecoder,
	)
}

func (s *SynchronizerOutputCreateSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerOutputCreateSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerOutputCreateSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerOutputCreateSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerOutputCreateSuite) TestCreateOutputs() {
	ctx := context.Background()

	// check setup
	proofCount := s.countOutputs(ctx)
	s.Require().Equal(0, proofCount)

	// first call
	err := s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)
	second := s.countOutputs(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerOutputCreateSuite) TestGetRawOutputRef() {
	outputs, err := s.rawNodeV2Repository.FindAllOutputsByFilter(s.ctx, FilterID{IDgt: 1})
	s.Require().NoError(err)
	rawOutput := outputs[0]
	rawOutputRef, err := s.synchronizerOutputCreate.GetRawOutputRef(rawOutput)
	s.Require().NoError(err)
	s.Equal("notice", rawOutputRef.Type)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, rawOutputRef.AppContract)
	s.Equal(0, int(rawOutputRef.InputIndex))
	s.Equal(false, rawOutputRef.HasProof)
	s.Equal(1, int(rawOutputRef.OutputIndex))
	s.Equal(2, int(rawOutputRef.RawID))
}

func (s *SynchronizerOutputCreateSuite) countOutputs(ctx context.Context) int {
	total, err := s.container.GetOutputRepository().CountAllOutputs(ctx)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerOutputCreateSuite) TestGetConvenienceVoucher() {
	outputs, err := s.rawNodeV2Repository.FindAllOutputsByFilter(s.ctx, FilterID{IDgt: 0})
	s.Require().NoError(err)
	rawOutput := outputs[0]
	rawOutputRef, err := s.synchronizerOutputCreate.GetRawOutputRef(rawOutput)
	s.Require().NoError(err)
	s.Equal("voucher", rawOutputRef.Type)
	cVoucher, err := s.synchronizerOutputCreate.GetConvenienceVoucher(rawOutput)
	s.Require().NoError(err)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, cVoucher.AppContract.Hex())
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", cVoucher.Destination.Hex())
	s.Equal(0, int(cVoucher.InputIndex))
	s.Equal(0, int(cVoucher.OutputIndex))
	s.Equal("3735928559", cVoucher.Value)
}

func (s *SynchronizerOutputCreateSuite) TestGetConvenienceNotice() {
	outputs, err := s.rawNodeV2Repository.FindAllOutputsByFilter(s.ctx, FilterID{IDgt: 1})
	s.Require().NoError(err)
	rawOutput := outputs[0]
	rawOutputRef, err := s.synchronizerOutputCreate.GetRawOutputRef(rawOutput)
	s.Require().NoError(err)
	s.Equal("notice", rawOutputRef.Type)
	cNotice, err := s.synchronizerOutputCreate.GetConvenienceNotice(rawOutput)
	s.Require().NoError(err)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, cNotice.AppContract)
	s.Equal(0, int(cNotice.InputIndex))
	s.Equal(1, int(cNotice.OutputIndex))
}
