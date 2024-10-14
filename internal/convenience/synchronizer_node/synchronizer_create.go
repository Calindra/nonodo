package synchronizernode

import (
	"context"

	"github.com/calindra/nonodo/internal/supervisor"
)

type SynchronizerCreateWorker struct{}

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	panic("unimplemented")
}

// String implements supervisor.Worker.
func (s SynchronizerCreateWorker) String() string {
	return "SynchronizerCreateWorker"
}

func NewSynchronizerCreateWorker() supervisor.Worker {
	return SynchronizerCreateWorker{}
}
