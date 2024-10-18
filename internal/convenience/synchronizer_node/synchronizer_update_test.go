package synchronizernode

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerUpdateNodeSuite struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	workerCtx                  context.Context
	timeoutCancel              context.CancelFunc
	workerCancel               context.CancelFunc
	workerResult               chan error
	synchronizerUpdateWorker   SynchronizerUpdateWorker
}

func (s *SynchronizerUpdateNodeSuite) SetupSuite() {
	timeout := 1 * time.Minute
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), timeout)

	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerUpdateNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	dbRawUrl := "postgres://postgres:password@localhost:5432/rollupsdb?sslmode=disable"

	var w supervisor.SupervisorWorker
	w.Name = "TestRawInputter"
	s.workerResult = make(chan error)

	// Temp
	tempDir, err := os.MkdirTemp("", "")
	s.NoError(err)
	s.tempDir = tempDir

	// Database
	sqliteFileName := filepath.Join(tempDir, "input.sqlite3")

	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	s.container = convenience.NewContainer(*db, false)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	rawInputRepository := s.container.GetRawInputRepository()
	s.synchronizerUpdateWorker = NewSynchronizerUpdateWorker(rawInputRepository, dbRawUrl)
	w.Workers = append(w.Workers, s.synchronizerUpdateWorker)

	// Supervisor
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

func (s *SynchronizerUpdateNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
	s.timeoutCancel()
}

func (s *SynchronizerUpdateNodeSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
	s.workerCancel()
}

func TestSynchronizerUpdateNodeSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerUpdateNodeSuite))
}

func (s *SynchronizerUpdateNodeSuite) TestSynchronizerUpdateInputStatus() {
	ctx := context.Background()
	err := s.container.GetRawInputRepository().Create(
		ctx, repository.RawInputRef{
			ID:          "1", //our ID
			RawID:       12,
			InputIndex:  1,
			AppContract: common.Address{}.Hex(),
			Status:      "NONE",
			ChainID:     "31337",
		},
	)
	s.Require().NoError(err)
	inputsStatusNone, err := s.synchronizerUpdateWorker.GetNextInputs2UpdateBatch(ctx)
	s.NoError(err)
	s.NotNil(inputsStatusNone)
}
