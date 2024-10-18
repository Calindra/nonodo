package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	workerCtx                  context.Context
	timeoutCancel              context.CancelFunc
	workerCancel               context.CancelFunc
	workerResult               chan error
	inputRepository            *repository.InputRepository
	inputRefRepository         *repository.RawInputRefRepository
}

func (s *SynchronizerNodeSuite) SetupSuite() {
	timeout := 1 * time.Minute
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), timeout)

	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	dbRawUrl := "postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"

	// var w supervisor.SupervisorWorker
	// w.Name = "TestRawInputter"
	s.workerResult = make(chan error)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "input.sqlite3")

	db, err := sqlx.ConnectContext(s.ctx, "sqlite3", sqliteFileName)
	s.NoError(err)

	s.inputRepository = &repository.InputRepository{Db: *db}
	err = s.inputRepository.CreateTables()
	s.NoError(err)
	s.inputRefRepository = &repository.RawInputRefRepository{Db: *db}
	err = s.inputRefRepository.CreateTables()
	s.NoError(err)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	wr := NewSynchronizerCreateWorker(s.inputRepository, s.inputRefRepository, dbRawUrl)

	// like Supervisor
	ready := make(chan struct{})
	go func() {
		s.workerResult <- wr.Start(s.workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("worker ready")
	}
}

func (s *SynchronizerNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
	s.timeoutCancel()
}

func (s *SynchronizerNodeSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
	s.workerCancel()
}

func XTestSynchronizerNodeSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerNodeSuite))
}

func (s *SynchronizerNodeSuite) TestSynchronizerNodeConnection() {
	val := <-s.workerResult
	s.NoError(val)
}
