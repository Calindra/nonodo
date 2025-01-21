package repository

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type SynchronizerRepositorySuite struct {
	suite.Suite
	repository *SynchronizerRepository
}

func (s *SynchronizerRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &SynchronizerRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	s.NoError(err)
}

func (s *SynchronizerRepositorySuite) TearDownTest() {
	err := s.repository.Db.Close()
	s.NoError(err)
}

func TestSynchronizerRepositorySuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerRepositorySuite))
}

func (s *SynchronizerRepositorySuite) TestCreateSyncFetch() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	count, err := s.repository.Count(ctx)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *SynchronizerRepositorySuite) TestGetLastFetched() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	_, err = s.repository.Create(ctx, &model.SynchronizerFetch{})
	s.NoError(err)
	lastFetch, err := s.repository.GetLastFetched(ctx)
	s.NoError(err)
	s.Equal(2, int(lastFetch.Id))
}

func (s *SynchronizerRepositorySuite) TestPurgeOldData() {
	ctx := context.Background()
	var (
		timeAfter  uint64 = 99
		timeBefore uint64 = timeAfter + 1
	)
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{
		TimestampAfter: timeAfter,
	})
	s.NoError(err)

	err = s.repository.PurgeData(ctx, timeBefore)
	s.NoError(err)

	count, err := s.repository.Count(ctx)
	s.NoError(err)
	s.Equal(1, int(count))
}
