package repository

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
	"github.com/lmittmann/tint"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type SynchronizerRepositorySuite struct {
	suite.Suite
	repository *SynchronizerRepository
}

func (s *SynchronizerRepositorySuite) SetupTest() {
	logOpts := new(tint.Options)
	logOpts.Level = slog.LevelDebug
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &SynchronizerRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	checkError(s.T(), err)
}

func TestSynchronizerRepositorySuiteSuite(t *testing.T) {
	suite.Run(t, new(SynchronizerRepositorySuite))
}

func (s *SynchronizerRepositorySuite) TestCreateSyncFetch() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	checkError(s.T(), err)
	count, err := s.repository.Count(ctx)
	checkError(s.T(), err)
	s.Equal(1, int(count))
}

func (s *SynchronizerRepositorySuite) TestGetLastFetched() {
	ctx := context.Background()
	_, err := s.repository.Create(ctx, &model.SynchronizerFetch{})
	checkError(s.T(), err)
	_, err = s.repository.Create(ctx, &model.SynchronizerFetch{})
	checkError(s.T(), err)
	lastFetch, err := s.repository.GetLastFetched(ctx)
	checkError(s.T(), err)
	s.Equal(2, int(lastFetch.Id))
}

func checkError(s *testing.T, err error) {
	if err != nil {
		s.Fatal(err.Error())
	}
}
