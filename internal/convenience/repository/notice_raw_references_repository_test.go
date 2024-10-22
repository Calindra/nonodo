package repository

import (
	"context"
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
	err = s.rawNoticeRefRepository.CreateTable()
	s.NoError(err)
}

func (s *RawNoticeRefSuite) TestRawRefNoticeCreateTables() {
	err := s.rawNoticeRefRepository.CreateTable()
	s.NoError(err)
}

func (s *RawNoticeRefSuite) TestRawRefNoticeShouldThrowAnErrorWhenThereIsNoTypeAttribute() {
	ctx := context.Background()

	rawNotice := RawNoticeRef{
		InputIndex:  1,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
	}

	err := s.rawNoticeRefRepository.Create(ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawNoticeRefSuite) TestRawRefNoticeShouldThrowAnErrorWhenTypeAttributeIsDiffFromVoucherOrNotice() {
	ctx := context.Background()

	rawNotice := RawNoticeRef{
		InputIndex:  1,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "report",
	}

	err := s.rawNoticeRefRepository.Create(ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawNoticeRefSuite) TestRawRefNoticeCreate() {
	ctx := context.Background()

	rawNotice := RawNoticeRef{
		InputIndex:  1,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "notice",
	}

	err := s.rawNoticeRefRepository.Create(ctx, rawNotice)
	s.NoError(err)

	var count int
	err = s.rawNoticeRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_output_raw_references WHERE input_index = ? AND app_contract = ? AND output_index = ?`,
		rawNotice.InputIndex, rawNotice.AppContract, rawNotice.OutputIndex).Scan(&count)

	s.NoError(err)
	s.Equal(1, count, "Expected one record to be inserted")
}
