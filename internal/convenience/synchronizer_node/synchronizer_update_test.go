package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

const TOTAL_INPUT_TEST = 65

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
	rawNode                    *RawNode
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
	s.rawNode = NewRawNode(dbRawUrl)
	rawInputRepository := s.container.GetRawInputRepository()
	s.synchronizerUpdateWorker = NewSynchronizerUpdateWorker(
		rawInputRepository,
		s.rawNode,
		s.container.GetInputRepository(),
	)
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

func (s *SynchronizerUpdateNodeSuite) TestGetNextInputBatch2Update() {
	ctx := context.Background()
	s.fillRefData(ctx)
	batchSize := 50
	s.synchronizerUpdateWorker.BatchSize = batchSize
	inputsStatusNone, err := s.synchronizerUpdateWorker.GetFirstRefWithStatusNone(ctx)
	s.NoError(err)
	s.NotNil(inputsStatusNone)
	s.Equal("NONE", inputsStatusNone.Status)
}

func (s *SynchronizerUpdateNodeSuite) TestUpdateInputStatusNotEqNone() {
	ctx := context.Background()
	s.fillRefData(ctx)
	batchSize := s.synchronizerUpdateWorker.BatchSize

	// first call
	err := s.synchronizerUpdateWorker.SyncInputStatus(ctx)
	s.Require().NoError(err)
	first := s.countAcceptedInput(ctx)
	s.Equal(50, batchSize)
	s.Equal(batchSize, first)

	// second call
	err = s.synchronizerUpdateWorker.SyncInputStatus(ctx)
	s.Require().NoError(err)
	second := s.countAcceptedInput(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerUpdateNodeSuite) countAcceptedInput(ctx context.Context) int {
	status := "Status"
	value := fmt.Sprintf("%d", model.CompletionStatusAccepted)
	filter := []*model.ConvenienceFilter{
		{
			Field: &status,
			Eq:    &value,
		},
	}
	total, err := s.container.GetInputRepository().Count(ctx, filter)
	s.Require().NoError(err)
	return int(total)
}

func (s *SynchronizerUpdateNodeSuite) fillRefData(ctx context.Context) {
	appContract := common.HexToAddress("0x5112cf49f2511ac7b13a032c4c62a48410fc28fb")
	msgSender := common.HexToAddress(devnet.SenderAddress)
	for i := 0; i < TOTAL_INPUT_TEST; i++ {
		id := strconv.FormatInt(int64(i), 10) // our ID
		err := s.container.GetRawInputRepository().Create(ctx, repository.RawInputRef{
			ID:          id,
			RawID:       uint64(i + 1),
			InputIndex:  uint64(i),
			AppContract: appContract.Hex(),
			Status:      "NONE",
			ChainID:     "31337",
		})
		s.Require().NoError(err)
		_, err = s.container.GetInputRepository().Create(ctx, model.AdvanceInput{
			ID:          id,
			Index:       i,
			Status:      model.CompletionStatusUnprocessed,
			AppContract: appContract,
			MsgSender:   msgSender,
		})
		s.Require().NoError(err)
	}
}
