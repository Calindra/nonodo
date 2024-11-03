package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
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
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	notice, err := s.repository.FindByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	s.Equal("0x0011", notice.Payload)
	s.Equal(1, int(notice.InputIndex))
	s.Equal(2, int(notice.OutputIndex))
}

func (s *NoticeRepositorySuite) TestCountNotices() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	_, err = s.repository.Create(ctx, &model.ConvenienceNotice{
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
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, err := s.repository.Create(ctx, &model.ConvenienceNotice{
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

func (s *NoticeRepositorySuite) TestGenerateBatchNoticeKey() {
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	var inputIndex uint64 = 0

	expectedKey := appContract.Hex() + "|0"

	result := GenerateBatchNoticeKey(appContract.Hex(), inputIndex)

	s.Equal(expectedKey, result)
}

func (s *NoticeRepositorySuite) TestBatchFindAll() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			_, err := s.repository.Create(
				ctx,
				&model.ConvenienceNotice{
					InputIndex:  uint64(i),
					OutputIndex: uint64(j),
					Payload:     "0x1122",
					AppContract: appContract.Hex(),
				})
			s.Require().NoError(err)
		}
	}
	filters := []*BatchFilterItemForNotice{
		{
			AppContract: appContract.Hex(),
			InputIndex:  0,
		},
	}
	results, err := s.repository.BatchFindAllNoticesByInputIndexAndAppContract(
		ctx, filters,
	)
	s.Require().Equal(0, len(err))
	s.Equal(1, len(results))
	s.Equal(4, len(results[0].Rows))
	s.Equal(4, int(results[0].Total))
}
