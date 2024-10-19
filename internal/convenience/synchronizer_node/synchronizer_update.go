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

func (s *SynchronizerUpdateWorker) GetFirstRefWithStatusNone(ctx context.Context) (*repository.RawInputRef, error) {
	return s.RawInputRefRepository.FindFirstInputByStatusNone(ctx, s.BatchSize)
}

func (s *SynchronizerUpdateWorker) FindFirst50RawInputsAfterRefWithStatus(
	ctx context.Context,
	inputRef repository.RawInputRef,
	status string,
) ([]RawInput, error) {
	return s.RawNode.FindAllInputsByFilter(ctx, FilterInput{
		IDgt:   inputRef.RawID,
		Status: status,
	}, &Pagination{
		Limit: uint64(s.BatchSize),
	})
}

func (s *SynchronizerUpdateWorker) FindAllRefsFor(ctx context.Context) {

}

func (s *SynchronizerUpdateWorker) StartTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawInputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, &db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerUpdateWorker) CommitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerUpdateWorker) MapIds(rawInputs []RawInput) []string {
	ids := make([]string, len(rawInputs))
	for i, input := range rawInputs {
		ids[i] = strconv.FormatUint(input.ID, 10)
	}
	return ids
}

type RosettaStatusRef struct {
	RawStatus string
	Status    model.CompletionStatus
}

func GetStatusRosetta() []RosettaStatusRef {
	return []RosettaStatusRef{
		{
			RawStatus: "ACCEPTED",
			Status:    model.CompletionStatusAccepted,
		},
		{
			RawStatus: "REJECTED",
			Status:    model.CompletionStatusRejected,
		},
		{
			RawStatus: "EXCEPTION",
			Status:    model.CompletionStatusException,
		},
		{
			RawStatus: "MACHINE_HALTED",
			Status:    model.CompletionStatusMachineHalted,
		},
		{
			RawStatus: "CYCLE_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusCycleLimitExceeded,
		},
		{
			RawStatus: "TIME_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusTimeLimitExceeded,
		},
		{
			RawStatus: "PAYLOAD_LENGTH_LIMIT_EXCEEDED",
			Status:    model.CompletionStatusPayloadLengthLimitExceeded,
		},
	}
}

// if we have a real ID it could be just one sql command using `id in (?)`
func (s *SynchronizerUpdateWorker) UpdateStatus(ctx context.Context, rawInputs []RawInput, status model.CompletionStatus) error {
	for _, rawInput := range rawInputs {
		appContract := common.BytesToAddress(rawInput.ApplicationAddress)
		slog.Debug("Update", "appContract", appContract, "index", rawInput.Index)
		err := s.InputRepository.UpdateStatus(ctx, appContract, rawInput.Index, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizerUpdateWorker) UpdateManyInputAndRefsStatus(ctx context.Context, rawInputs []RawInput, rosetta RosettaStatusRef) error {
	err := s.RawInputRefRepository.UpdateStatus(ctx, s.MapIds(rawInputs), rosetta.RawStatus)
	if err != nil {
		return err
	}
	err = s.UpdateStatus(ctx, rawInputs, rosetta.Status)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerUpdateWorker) SyncInputStatus(ctx context.Context) error {
	ctxWithTx, err := s.StartTransaction(ctx)
	if err != nil {
		return err
	}
	inputRef, err := s.GetFirstRefWithStatusNone(ctxWithTx)
	if err != nil {
		return err
	}
	if inputRef != nil {
		rosettaStone := GetStatusRosetta()
		for _, rosetta := range rosettaStone {
			rawInputs, err := s.FindFirst50RawInputsAfterRefWithStatus(ctx, *inputRef, rosetta.RawStatus)
			if err != nil {
				return err
			}
			err = s.UpdateManyInputAndRefsStatus(ctxWithTx, rawInputs, rosetta)
			if err != nil {
				return err
			}
		}
	}
	err = s.CommitTransaction(ctxWithTx)
	if err != nil {
		return err
	}
	return nil
}
