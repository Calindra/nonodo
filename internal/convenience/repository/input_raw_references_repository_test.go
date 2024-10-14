package repository

import (
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
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

func (s *RawInputRefSuite) TestCreateTables() {
}
