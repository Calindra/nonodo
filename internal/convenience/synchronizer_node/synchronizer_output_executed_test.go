package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerOutputExecutedSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerOutputExecuted *SynchronizerOutputExecuted
	rawNodeV2Repository        *RawRepository
	synchronizerOutputCreate   *SynchronizerOutputCreate
	dbNodeV2                   *sqlx.DB
}

func (s *SynchronizerOutputExecutedSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerOutputExecutedSuite) SetupTest() {
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

	s.dbNodeV2 = sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNodeV2Repository = NewRawRepository(RAW_DB_URL, s.dbNodeV2)

	s.synchronizerOutputExecuted = NewSynchronizerOutputExecuted(
		s.container.GetVoucherRepository(),
		s.container.GetNoticeRepository(),
		s.rawNodeV2Repository,
		s.container.GetRawOutputRefRepository(),
	)

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

func (s *SynchronizerOutputExecutedSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerOutputExecutedSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerOutputExecutedSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerOutputExecutedSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerOutputExecutedSuite) TestUpdateOutputs() {
	ctx := context.Background()

	_, err := s.dbNodeV2.ExecContext(ctx, `
		UPDATE output 
		SET transaction_hash = NULL,
			updated_at = '2024-11-06 15:30:00'
	`)
	s.Require().NoError(err)

	// check setup
	outputCount := s.countOutputs(ctx)
	s.Require().Equal(0, outputCount)

	// first call
	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)
	second := s.countOutputs(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)

	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)
	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)
	err = s.synchronizerOutputCreate.SyncOutputs(ctx)
	s.Require().NoError(err)

	// check setup
	executedCount := s.countExecuted(ctx)
	s.Require().Equal(0, executedCount)

	_, err = s.dbNodeV2.ExecContext(ctx, `
		UPDATE output 
		SET
			transaction_hash = '\x0011223344',
			updated_at = NOW()
		WHERE substring(raw_data FROM 1 FOR 2) = '\x237A'
	`)
	s.Require().NoError(err)

	// first call
	err = s.synchronizerOutputExecuted.SyncOutputsExecution(ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerOutputExecuted.SyncOutputsExecution(ctx)
	s.Require().NoError(err)
	err = s.synchronizerOutputExecuted.SyncOutputsExecution(ctx)
	s.Require().NoError(err)
	second = s.countExecuted(ctx)
	s.Equal(TOTAL_INPUT_TEST+1, second)
	err = s.synchronizerOutputExecuted.SyncOutputsExecution(ctx)
	s.Require().NoError(err)
	lastCount := s.countExecuted(ctx)
	s.Equal(TOTAL_INPUT_TEST+1, lastCount)
	// s.Fail("uncomment this line just to see the logs")
}

func (s *SynchronizerOutputExecutedSuite) countOutputs(ctx context.Context) int {
	total, err := s.container.GetOutputRepository().CountAllOutputs(ctx)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerOutputExecutedSuite) countExecuted(ctx context.Context) int {
	field := model.EXECUTED
	value := "true"
	filters := []*model.ConvenienceFilter{
		{
			Field: &field,
			Eq:    &value,
		},
	}
	total, err := s.container.GetVoucherRepository().Count(ctx, filters)
	s.Require().NoError(err)
	return int(total)
}
