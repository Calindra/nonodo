package synchronizernode

import (
	"context"

	"github.com/calindra/nonodo/internal/supervisor"
)

type SynchronizerUpdateWorker struct {
	DbRawUrl string
	DbRaw    *RawNode
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

func NewSynchronizerUpdateWorker(dbRawUrl string) supervisor.Worker {
	return SynchronizerUpdateWorker{DbRawUrl: dbRawUrl}
}
