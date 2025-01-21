package synchronizernode

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

const DefaultBatchSize = 50

type SynchronizerUpdate struct {
	DbRawUrl              string
	RawNode               *RawRepository
	RawInputRefRepository *repository.RawInputRefRepository
	InputRepository       *repository.InputRepository
	BatchSize             int
}

func NewSynchronizerUpdate(
	rawInputRefRepository *repository.RawInputRefRepository,
	rawNode *RawRepository,
	inputRepository *repository.InputRepository,
) SynchronizerUpdate {
	return SynchronizerUpdate{
		RawNode:               rawNode,
		RawInputRefRepository: rawInputRefRepository,
		BatchSize:             DefaultBatchSize,
		InputRepository:       inputRepository,
	}
}

func (s *SynchronizerUpdate) getFirstRefWithStatusNone(ctx context.Context) (*repository.RawInputRef, error) {
	return s.RawInputRefRepository.FindFirstInputByStatusNone(ctx, s.BatchSize)
}

func (s *SynchronizerUpdate) findFirst50RawInputsAfterRefWithStatus(
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

func (s *SynchronizerUpdate) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawInputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, &db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerUpdate) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerUpdate) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}

func (s *SynchronizerUpdate) mapIds(rawInputs []RawInput) []string {
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
func (s *SynchronizerUpdate) updateStatus(ctx context.Context, rawInputs []RawInput, status model.CompletionStatus) error {
	for _, rawInput := range rawInputs {
		appContract := common.BytesToAddress(rawInput.ApplicationAddress)
		// slog.Debug("Update", "appContract", appContract, "index", rawInput.Index, "status", status)
		err := s.InputRepository.UpdateStatus(ctx, appContract, rawInput.Index, status)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizerUpdate) updateManyInputAndRefsStatus(ctx context.Context, rawInputs []RawInput, rosetta RosettaStatusRef) error {
	err := s.RawInputRefRepository.UpdateStatus(ctx, s.mapIds(rawInputs), rosetta.RawStatus)
	if err != nil {
		return err
	}
	err = s.updateStatus(ctx, rawInputs, rosetta.Status)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerUpdate) SyncInputStatus(ctx context.Context) error {
	ctxWithTx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	inputRef, err := s.getFirstRefWithStatusNone(ctxWithTx)
	if err != nil {
		return err
	}
	if inputRef != nil {
		rosettaStone := GetStatusRosetta()
		for _, rosetta := range rosettaStone {
			rawInputs, err := s.findFirst50RawInputsAfterRefWithStatus(ctx, *inputRef, rosetta.RawStatus)
			if err != nil {
				s.rollbackTransaction(ctxWithTx)
				return err
			}
			err = s.updateManyInputAndRefsStatus(ctxWithTx, rawInputs, rosetta)
			if err != nil {
				s.rollbackTransaction(ctxWithTx)
				return err
			}
		}
	}
	err = s.commitTransaction(ctxWithTx)
	if err != nil {
		return err
	}
	return nil
}
