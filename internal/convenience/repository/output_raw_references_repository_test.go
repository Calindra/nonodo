package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/stretchr/testify/suite"
)

type RawOutputRefSuite struct {
	suite.Suite
	noticeRepository       *NoticeRepository
	rawOutputRefRepository *RawOutputRefRepository
	dbFactory              *commons.DbFactory
}

func (s *RawOutputRefSuite) TearDownTest() {
	defer s.dbFactory.Cleanup()
}

func TestRawRefOutputSuite(t *testing.T) {
	suite.Run(t, new(RawOutputRefSuite))
}

func (s *RawOutputRefSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("input.sqlite3")
	s.noticeRepository = &NoticeRepository{
		Db: *db,
	}
	s.rawOutputRefRepository = &RawOutputRefRepository{
		Db: db,
	}

	err := s.noticeRepository.CreateTables()
	s.NoError(err)
	err = s.rawOutputRefRepository.CreateTable()
	s.NoError(err)
}

func (s *RawOutputRefSuite) TestRawRefOutputCreateTables() {
	err := s.rawOutputRefRepository.CreateTable()
	s.NoError(err)
}

func (s *RawOutputRefSuite) TestRawRefOutputShouldThrowAnErrorWhenThereIsNoTypeAttribute() {
	ctx := context.Background()

	rawNotice := RawOutputRef{
		InputIndex:  1,
		RawID:       2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
	}

	err := s.rawOutputRefRepository.Create(ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawOutputRefSuite) TestRawRefOutputShouldThrowAnErrorWhenTypeAttributeIsDiffFromVoucherOrNotice() {
	ctx := context.Background()

	rawNotice := RawOutputRef{
		InputIndex:  1,
		RawID:       2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "report",
	}

	err := s.rawOutputRefRepository.Create(ctx, rawNotice)
	s.ErrorContains(err, "sqlite3: constraint failed: CHECK constraint failed: type IN ('voucher', 'notice')")
}

func (s *RawOutputRefSuite) TestRawRefOutputCreate() {
	ctx := context.Background()

	rawOutput := RawOutputRef{
		InputIndex:  1,
		RawID:       2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "notice",
	}

	err := s.rawOutputRefRepository.Create(ctx, rawOutput)
	s.NoError(err)

	var count int
	err = s.rawOutputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_output_raw_references WHERE input_index = ? AND app_contract = ? AND output_index = ?`,
		rawOutput.InputIndex, rawOutput.AppContract, rawOutput.OutputIndex).Scan(&count)

	s.NoError(err)
	s.Equal(1, count)
}

func (s *RawOutputRefSuite) TestRawRefOutputGetLatestId() {
	ctx := context.Background()

	firstRawOutput := RawOutputRef{
		RawID:       1,
		InputIndex:  1,
		AppContract: "0x123456789abcdef",
		OutputIndex: 2,
		Type:        "notice",
	}

	err := s.rawOutputRefRepository.Create(ctx, firstRawOutput)
	s.NoError(err)

	lastRawOutput := RawOutputRef{
		RawID:       2,
		InputIndex:  2,
		AppContract: "0x123456789abcdef",
		OutputIndex: 23,
		Type:        "voucher",
	}

	err = s.rawOutputRefRepository.Create(ctx, lastRawOutput)
	s.NoError(err)

	var count int
	err = s.rawOutputRefRepository.Db.QueryRow(`SELECT COUNT(*) FROM convenience_output_raw_references`).Scan(&count)
	s.NoError(err)
	//check if there are two records in the table.
	s.Equal(2, count)

	outputId, err := s.rawOutputRefRepository.GetLatestOutputRawId(ctx)
	s.NoError(err)
	s.Equal(lastRawOutput.RawID, outputId)
}
