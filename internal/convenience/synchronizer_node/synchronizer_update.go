package synchronizernode

import (
	"context"

	"github.com/calindra/nonodo/internal/supervisor"
)

type SynchronizerUpdateWorker struct{}

// Start implements supervisor.Worker.
func (s SynchronizerUpdateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	panic("unimplemented")
}

// String implements supervisor.Worker.
func (s SynchronizerUpdateWorker) String() string {
	return "SynchronizerUpdateWorker"
}

func NewSynchronizerUpdateWorker() supervisor.Worker {
	return SynchronizerUpdateWorker{}
}
