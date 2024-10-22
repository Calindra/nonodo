package repository

import (
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/stretchr/testify/suite"
)

type RawNoticeRefSuite struct {
	suite.Suite
	noticeRepository       *NoticeRepository
	rawNoticeRefRepository *RawNoticeRefRepository
	dbFactory              *commons.DbFactory
}

func (s *RawNoticeRefSuite) TearDownTest() {
	defer s.dbFactory.Cleanup()
}

func TestRawRefNoticeSuite(t *testing.T) {
	suite.Run(t, new(RawNoticeRefSuite))
}

func (s *RawNoticeRefSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("input.sqlite3")
	s.noticeRepository = &NoticeRepository{
		Db: *db,
	}
	s.rawNoticeRefRepository = &RawNoticeRefRepository{
		Db: *db,
	}

	err := s.noticeRepository.CreateTables()
	s.NoError(err)
	err = s.rawNoticeRefRepository.CreateTables()
	s.NoError(err)
}

func (s *RawNoticeRefSuite) TestRawRefNoticeCreateTables() {
	err := s.rawNoticeRefRepository.CreateTables()
	s.NoError(err)
}
