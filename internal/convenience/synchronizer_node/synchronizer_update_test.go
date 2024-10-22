package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/devnet"
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
	synchronizerUpdate         SynchronizerUpdate
	rawNode                    *RawRepository
}

func (s *SynchronizerUpdateNodeSuite) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerUpdateNodeSuite) SetupTest() {
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
	rawInputRefRepository := s.container.GetRawInputRepository()
	s.synchronizerUpdate = NewSynchronizerUpdate(
		rawInputRefRepository,
		s.rawNode,
		s.container.GetInputRepository(),
	)
}

func (s *SynchronizerUpdateNodeSuite) TearDownSuite() {
	if s.dockerComposeStartedByTest {
		err := raw.StopDockerCompose(s.ctx)
		s.NoError(err)
	}
}

func (s *SynchronizerUpdateNodeSuite) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerUpdateNodeSuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerUpdateNodeSuite))
}

func (s *SynchronizerUpdateNodeSuite) TestGetFirstRefWithStatusNone() {
	ctx := context.Background()
	s.fillRefData(ctx)
	batchSize := 50
	s.synchronizerUpdate.BatchSize = batchSize
	inputsStatusNone, err := s.synchronizerUpdate.getFirstRefWithStatusNone(ctx)
	s.NoError(err)
	s.NotNil(inputsStatusNone)
	s.Equal("NONE", inputsStatusNone.Status)
}

// Dear Programmer, I hope this message finds you well.
// Keep coding, keep learning, and never forget—your work shapes the future.
func (s *SynchronizerUpdateNodeSuite) TestUpdateInputStatusNotEqNone() {
	ctx := context.Background()
	s.fillRefData(ctx)

	// check setup
	unprocessed := s.countInputWithStatusNone(ctx)
	s.Require().Equal(TOTAL_INPUT_TEST, unprocessed)

	batchSize := s.synchronizerUpdate.BatchSize

	// first call
	err := s.synchronizerUpdate.SyncInputStatus(ctx)
	s.Require().NoError(err)
	first := s.countAcceptedInput(ctx)
	s.Equal(50, batchSize)
	s.Equal(batchSize, first)

	// second call
	err = s.synchronizerUpdate.SyncInputStatus(ctx)
	s.Require().NoError(err)
	second := s.countAcceptedInput(ctx)
	s.Equal(TOTAL_INPUT_TEST, second)
}

func (s *SynchronizerUpdateNodeSuite) countInputWithStatusNone(ctx context.Context) int {
	status := model.STATUS_PROPERTY
	value := fmt.Sprintf("%d", model.CompletionStatusUnprocessed)
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

func (s *SynchronizerUpdateNodeSuite) countAcceptedInput(ctx context.Context) int {
	status := model.STATUS_PROPERTY
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
