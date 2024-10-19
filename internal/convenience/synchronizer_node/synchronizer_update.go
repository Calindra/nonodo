package synchronizernode

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

const DefaultBatchSize = 50

type SynchronizerUpdateWorker struct {
	DbRawUrl              string
	RawNode               *RawNode
	RawInputRefRepository *repository.RawInputRefRepository
	InputRepository       *repository.InputRepository
	BatchSize             int
}

// Start implements supervisor.Worker.
func (s SynchronizerUpdateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	return nil
}

// String implements supervisor.Worker.
func (s SynchronizerUpdateWorker) String() string {
	return "SynchronizerUpdateWorker"
}

func NewSynchronizerUpdateWorker(
	rawInputRefRepository *repository.RawInputRefRepository,
	rawNode *RawNode,
	inputRepository *repository.InputRepository,
) SynchronizerUpdateWorker {
	return SynchronizerUpdateWorker{
		RawNode:               rawNode,
		RawInputRefRepository: rawInputRefRepository,
		BatchSize:             DefaultBatchSize,
		InputRepository:       inputRepository,
	}
}

func (s *SynchronizerUpdateWorker) GetNextInputBatch2Update(ctx context.Context) ([]repository.RawInputRef, error) {
	return s.RawInputRefRepository.FindInputsByStatusNone(ctx, s.BatchSize)
}

func (s *SynchronizerUpdateWorker) UpdateInputStatusNotEqNone(ctx context.Context) error {
	refs, err := s.GetNextInputBatch2Update(ctx)
	if err != nil {
		return err
	}
	for _, inputRef := range refs {
		inputs, err := s.RawNode.FindAllInputsByFilter(ctx, FilterInput{
			IDgt:     inputRef.RawID,
			StatusNe: "NONE",
		}, &Pagination{
			Limit: uint64(s.BatchSize),
		})
		if err != nil {
			return err
		}
		for _, rawInput := range inputs {
			appContract := common.BytesToAddress(rawInput.ApplicationAddress)
			slog.Debug("Update", "appContract", appContract, "id", rawInput.ID)
			txId := strconv.FormatUint(rawInput.ID, 10)
			err := s.InputRepository.UpdateStatus(ctx, appContract, txId, model.CompletionStatusAccepted)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
