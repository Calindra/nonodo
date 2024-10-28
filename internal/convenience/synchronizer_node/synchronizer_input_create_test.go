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
	"github.com/calindra/nonodo/postgres/raw"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type SynchronizerInputCreate struct {
	suite.Suite
	ctx                        context.Context
	dockerComposeStartedByTest bool
	tempDir                    string
	container                  *convenience.Container
	synchronizerInputCreate    *SynchronizerCreateInput
	rawNodeV2Repository        *RawRepository
}

func (s *SynchronizerInputCreate) SetupSuite() {
	pgUp := commons.IsPortInUse(5432)
	if !pgUp {
		err := raw.RunDockerCompose(s.ctx)
		s.NoError(err)
		s.dockerComposeStartedByTest = true
	}
}

func (s *SynchronizerInputCreate) SetupTest() {
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

	abi, err := contracts.InputsMetaData.GetAbi()
	if err != nil {
		s.Require().NoError(err)
	}
	abiDecoder := NewAbiDecoder(abi)
	s.synchronizerInputCreate = NewSynchronizerCreateInput(
		s.container.GetInputRepository(),
		s.container.GetRawInputRepository(),
		s.rawNodeV2Repository,
		abiDecoder,
	)
}

func (s *SynchronizerInputCreate) TearDownTest() {
	defer os.RemoveAll(s.tempDir)
}

func TestSynchronizerInputCreateSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerInputCreate))
}

func (s *SynchronizerInputCreate) TestGetAdvanceInputFromMap() {
	inputs, err := s.rawNodeV2Repository.FindAllInputsByFilter(s.ctx, FilterInput{IDgt: 1}, &Pagination{Limit: 1})
	s.Require().NoError(err)

	rawInput := inputs[0]
	advanceInput, err := s.synchronizerInputCreate.GetAdvanceInputFromMap(rawInput)
	s.Require().NoError(err)
	s.Equal("0", advanceInput.ID)
	s.Equal(DEFAULT_TEST_APP_CONTRACT, advanceInput.AppContract.Hex())
	s.Equal(0, advanceInput.Index)
	s.Equal(0, advanceInput.InputBoxIndex)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", advanceInput.MsgSender.Hex())
	s.Equal(uint64(0x7a), advanceInput.BlockNumber)
	s.Equal("31337", advanceInput.ChainId)
	s.Equal(commons.ConvertStatusStringToCompletionStatus("ACCEPTED"), advanceInput.Status)
}