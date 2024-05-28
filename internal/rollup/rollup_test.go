package rollup

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const TestTimeout = 5 * time.Second

//
// Test Suite
//

type IInputRepository interface {
	// CreateTables() error
	// Create(input AdvanceInput) (*AdvanceInput, error)
	// Update(input AdvanceInput) (*AdvanceInput, error)
	// FindByStatusNeDesc(status CompletionStatus) (*AdvanceInput, error)
	// FindByStatus(status CompletionStatus) (*AdvanceInput, error)
	// FindByIndex(index int) (*AdvanceInput, error)
	// Count(filter []*model.ConvenienceFilter) (uint64, error)
	// FindAll(
	// 	first *int,
	// 	last *int,
	// 	after *string,
	// 	before *string,
	// 	filter []*model.ConvenienceFilter,
	// ) (*commons.PageResult[AdvanceInput], error)
}

type RollupSuite struct {
	suite.Suite
	ctx             context.Context
	cancel          context.CancelFunc
	inputRepository IInputRepository
}

type InputRepositoryMock struct {
	mock.Mock
}

func (s *RollupSuite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), TestTimeout)
	commons.ConfigureLog(slog.LevelDebug)
	s.inputRepository = &InputRepositoryMock{}
}

func TestRollupSuite(t *testing.T) {
	suite.Run(t, new(RollupSuite))
}

func (s *RollupSuite) teardown() {
	// nothing to do
	select {
	case <-s.ctx.Done():
		s.T().Error(s.ctx.Err())
	default:
		s.cancel()
	}
}

func (s *RollupSuite) TestFetcher() {
	defer s.teardown()

}
