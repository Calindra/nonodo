package synchronizernode

import (
	"context"

	"github.com/calindra/nonodo/internal/convenience/repository"
)

type SynchronizerUpdateWorker struct {
	DbRawUrl              string
	DbRaw                 *RawNode
	RawInputRefRepository *repository.RawInputRefRepository
}

// Start implements supervisor.Worker.
func (s SynchronizerUpdateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}

	s.DbRaw = NewRawNode(s.DbRawUrl)
	db, err := s.DbRaw.Connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	return nil
}

// String implements supervisor.Worker.
func (s SynchronizerUpdateWorker) String() string {
	return "SynchronizerUpdateWorker"
}

func NewSynchronizerUpdateWorker(container *repository.RawInputRefRepository, dbRawUrl string) SynchronizerUpdateWorker {
	return SynchronizerUpdateWorker{DbRawUrl: dbRawUrl, RawInputRefRepository: container}
}

func (s *SynchronizerUpdateWorker) GetNextInputs2UpdateBatch(ctx context.Context) (*[]repository.RawInputRef, error) {
	batchSize := 10
	return s.RawInputRefRepository.FindInputsByStatusNone(ctx, batchSize)
}
