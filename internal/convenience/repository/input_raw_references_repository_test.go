package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	configtest "github.com/calindra/nonodo/internal/convenience/config_test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type RawInputRefSuite struct {
	suite.Suite
	inputRepository       *InputRepository
	RawInputRefRepository *RawInputRefRepository
	dbFactory             *commons.DbFactory
}

func TestRawRefInputSuite(t *testing.T) {
	suite.Run(t, new(RawInputRefSuite))
}

func (s *RawInputRefSuite) TearDownTest() {
	defer s.dbFactory.Cleanup()
}

func (s *RawInputRefSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("input.sqlite3")
	s.inputRepository = &InputRepository{
		Db: *db,
	}
	s.RawInputRefRepository = &RawInputRefRepository{
		Db: *db,
	}

	err := s.inputRepository.CreateTables()
	s.NoError(err)
	err = s.RawInputRefRepository.CreateTables()
	s.NoError(err)
}

func (s *RawInputRefSuite) TestNoDuplicateInputs() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		RawID:       uint64(1),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	err = s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		RawID:       uint64(1),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})
	s.Require().NoError(err)

	var count int
	err = s.RawInputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_input_raw_references WHERE raw_id = ? AND app_contract = ?`,
		uint64(1), appContract.Hex()).Scan(&count)

	s.Require().NoError(err)
	s.Require().Equal(1, count)
}

func (s *RawInputRefSuite) TestSaveDifferentsInputs() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		RawID:       uint64(1),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	err = s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "002",
		RawID:       uint64(2),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})
	s.Require().NoError(err)

	var count int
	err = s.RawInputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_input_raw_references`).Scan(&count)

	s.Require().NoError(err)
	s.Require().Equal(2, count)
}

func (s *RawInputRefSuite) TestFindByRawIdAndAppContract() {
	ctx := context.Background()
	appContract := common.HexToAddress(configtest.DEFAULT_TEST_APP_CONTRACT)
	err := s.RawInputRefRepository.Create(ctx, RawInputRef{
		ID:          "001",
		RawID:       uint64(1),
		InputIndex:  uint64(1),
		AppContract: appContract.Hex(),
		Status:      "NONE",
		ChainID:     "31337",
	})

	s.Require().NoError(err)

	input, err := s.RawInputRefRepository.FindByRawIdAndAppContract(ctx, uint64(1), &appContract)

	s.Require().NoError(err)
	s.Require().Equal("001", input.ID)
	s.Require().Equal("NONE", input.Status)
	s.Require().Equal("31337", input.ChainID)
	s.Require().Equal(appContract.Hex(), input.AppContract)
}
