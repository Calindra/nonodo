package synchronizer

import (
	"context"
	"log/slog"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
)

type CleanSynchronizer struct {
	SynchronizerRepository *repository.SynchronizerRepository
	Period                 time.Duration
}

func NewCleanSynchronizer(
	SynchronizerRepository *repository.SynchronizerRepository,
	period *time.Duration,
) *CleanSynchronizer {
	var Period time.Duration = 10 * time.Minute

	if period != nil {
		Period = *period
	}

	return &CleanSynchronizer{SynchronizerRepository: SynchronizerRepository, Period: Period}
}

func (x CleanSynchronizer) String() string {
	return "CleanSynchronizer"
}

func (x CleanSynchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}

	periodMili := x.Period.Milliseconds()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(x.Period):
			timestampBefore := uint64(time.Now().UnixMilli() - periodMili)
			slog.Debug("Cleaning synchronizer", "timestampBefore", timestampBefore)
			err := x.SynchronizerRepository.PurgeData(ctx, timestampBefore)
			if err != nil {
				slog.Error("Error purging data", "Error", err)
			}
		}
	}

}
