package synchronizernode

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	workerCtx                  context.Context
	timeoutCancel              context.CancelFunc
	workerCancel               context.CancelFunc
	workerResult               chan error
	inputRepository            *repository.InputRepository
	inputRefRepository         *repository.RawInputRefRepository
	dbFactory                  *commons.DbFactory
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

	s.workerResult = make(chan error)

	// Database
	s.dbFactory = commons.NewDbFactory()
	db, err := s.dbFactory.CreateDbCtx(s.ctx, "input.sqlite3")
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
	s.dbFactory.Cleanup()
	s.workerCancel()
}

func TestSynchronizerNodeSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerNodeSuite))
}

func (s *SynchronizerNodeSuite) XTestSynchronizerNodeConnection() {
	val := <-s.workerResult
	s.NoError(val)
}
