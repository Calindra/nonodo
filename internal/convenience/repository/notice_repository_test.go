package repository

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type NoticeRepositorySuite struct {
	suite.Suite
	repository *NoticeRepository
}

func (s *NoticeRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &NoticeRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	s.NoError(err)
}

func TestNoticeRepositorySuite(t *testing.T) {
	suite.Run(t, new(NoticeRepositorySuite))
}

func (s *NoticeRepositorySuite) TestCreateNotice() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *NoticeRepositorySuite) TestFindByInputAndOutputIndex() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	notice, err := s.repository.FindByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	fmt.Println(notice.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", notice.Destination.String())
	s.Equal("0x0011", notice.Payload)
	s.Equal(1, int(notice.InputIndex))
	s.Equal(2, int(notice.OutputIndex))
}

func (s *NoticeRepositorySuite) TestCountNotices() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	_, err = s.repository.Create(ctx, &model.ConvenienceNotice{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 0,
	})
	s.NoError(err)
	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(2, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "InputIndex"
		value := "2"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	total, err = s.repository.Count(ctx, filters)
	s.NoError(err)
	s.Equal(1, int(total))
}

func (s *NoticeRepositorySuite) TestNoticePagination() {
	destination := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
			Destination: destination,
			Payload:     "0x0011",
			InputIndex:  uint64(i),
			OutputIndex: 0,
		})
		s.NoError(err)
	}

	total, err := s.repository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(30, int(total))

	filters := []*model.ConvenienceFilter{}
	first := 10
	notices, err := s.repository.FindAllNotices(ctx, &first, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(0, int(notices.Rows[0].InputIndex))
	s.Equal(9, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	after := commons.EncodeCursor(10)
	notices, err = s.repository.FindAllNotices(ctx, &first, nil, &after, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(11, int(notices.Rows[0].InputIndex))
	s.Equal(20, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	last := 10
	notices, err = s.repository.FindAllNotices(ctx, nil, &last, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(20, int(notices.Rows[0].InputIndex))
	s.Equal(29, int(notices.Rows[len(notices.Rows)-1].InputIndex))

	before := commons.EncodeCursor(20)
	notices, err = s.repository.FindAllNotices(ctx, nil, &last, nil, &before, filters)
	s.NoError(err)
	s.Equal(10, len(notices.Rows))
	s.Equal(10, int(notices.Rows[0].InputIndex))
	s.Equal(19, int(notices.Rows[len(notices.Rows)-1].InputIndex))
}

func (s *NoticeRepositorySuite) TestWrongAddress() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	filters := []*model.ConvenienceFilter{}
	{
		field := model.DESTINATION
		value := "0xError"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	_, err = s.repository.FindAllNotices(ctx, nil, nil, nil, nil, filters)
	if err == nil {
		s.Fail("where is the error?")
	}
	s.Equal("wrong address value", err.Error())
}
