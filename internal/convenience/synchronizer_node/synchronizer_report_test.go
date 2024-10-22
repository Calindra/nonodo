package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerReportSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerReport         *SynchronizerReport
	rawNode                    *RawRepository
}

func (s *SynchronizerReportSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerReportSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "input.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(*db, false)

	dbNodeV2 := sqlx.MustConnect("postgres", RAW_DB_URL)
	s.rawNode = NewRawRepository(RAW_DB_URL, dbNodeV2)
	s.synchronizerReport = NewSynchronizerReport(
		s.container.GetReportRepository(),
		s.rawNode,
	)
}

func (s *SynchronizerReportSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerReportSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerReportSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerReportSuite))
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerReportSuite) TestUpdateInputStatusNotEqNone() {
	ctx := context.Background()

	// check setup
	startReportCount := s.countHLReports(ctx)
	s.Require().Equal(0, startReportCount)

	// first call
	err := s.synchronizerReport.SyncReports(ctx)
	s.Require().NoError(err)

	// second call
	err = s.synchronizerReport.SyncReports(ctx)
	s.Require().NoError(err)
	second := s.countHLReports(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerReportSuite) countHLReports(ctx context.Context) int {
	total, err := s.container.GetReportRepository().Count(ctx, nil)
	s.Require().NoError(err)
	return int(total)
}
