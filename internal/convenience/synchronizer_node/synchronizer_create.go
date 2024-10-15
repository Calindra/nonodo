package synchronizernode

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type SynchronizerCreateWorker struct {
	Container *convenience.Container
	DbRawUrl  string
	DbRaw     *RawNode
}

const DEFAULT_TIMEOUT = 1 * time.Second

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	s.DbRaw = NewRawNode(s.DbRawUrl)
	db, err := s.DbRaw.Connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	return s.WatchNewInputs(ctx, db)
}

func (s SynchronizerCreateWorker) WatchNewInputs(stdCtx context.Context, db *sqlx.DB) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	rawInputRep := s.Container.GetRawInputRepository()
	latestRawID, err := rawInputRep.GetLatestRawId(ctx)

	if err != nil {
		return err
	}

	for {
		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
					inputs, err := s.DbRaw.FindAllInputsByFilter(ctx, FilterInput{IDgt: latestRawID})
					if err != nil {
						errCh <- err
					}

					for _, input := range inputs {
						rawInputRef := repository.RawInputRef{
							RawID:       uint64(input.ID),
							InputIndex:  input.Index,
							AppContract: common.BytesToAddress(input.ApplicationAddress).Hex(),
							Status:      input.Status,
							ChainID:     "",
						}

						err := rawInputRep.Create(ctx, rawInputRef)
						if err != nil {
							errCh <- err
						}

						rawInputRefID, err := strconv.ParseUint(rawInputRef.ID, 10, 64)
						if err != nil {
							errCh <- err
						}
						latestRawID = rawInputRefID
					}
				}
			}
		}()

		wrong := <-errCh

		if wrong != nil {
			return wrong
		}

		slog.Debug("Retrying to fetch new inputs")
		time.Sleep(DEFAULT_TIMEOUT)
	}
}

// String implements supervisor.Worker.
func (s SynchronizerCreateWorker) String() string {
	return "SynchronizerCreateWorker"
}

func NewSynchronizerCreateWorker(container *convenience.Container, dbRawUrl string) supervisor.Worker {
	return SynchronizerCreateWorker{Container: container, DbRawUrl: dbRawUrl}
}
