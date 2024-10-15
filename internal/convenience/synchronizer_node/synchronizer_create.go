package synchronizernode

import (
	"context"

	"github.com/calindra/nonodo/internal/supervisor"
)

type SynchronizerCreateWorker struct {
	DbRawUrl string
	DbRaw    *RawNode
}

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
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
func (s SynchronizerCreateWorker) String() string {
	return "SynchronizerCreateWorker"
}

func NewSynchronizerCreateWorker(dbRawUrl string) supervisor.Worker {
	return SynchronizerCreateWorker{DbRawUrl: dbRawUrl}
}
