package model

import (
	"context"

	cModel "github.com/calindra/nonodo/internal/convenience/model"
)

type Sequencer interface {
	FinishAndGetNext(accept bool) (cModel.Input, error)
}

type InputBoxSequencer struct {
	model *NonodoModel
}

func NewInputBoxSequencer(model *NonodoModel) *InputBoxSequencer {
	return &InputBoxSequencer{model: model}
}

func NewEspressoSequencer(model *NonodoModel) *EspressoSequencer {
	return &EspressoSequencer{model: model}
}

func (ibs *InputBoxSequencer) FinishAndGetNext(accept bool) (cModel.Input, error) {
	return FinishAndGetNext(ibs.model, accept)
}

func (es *EspressoSequencer) FinishAndGetNext(accept bool) (cModel.Input, error) {
	return FinishAndGetNext(es.model, accept)
}

type EspressoSequencer struct {
	model *NonodoModel
}

func FinishAndGetNext(m *NonodoModel, accept bool) (cModel.Input, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// finish current input
	var status cModel.CompletionStatus
	if accept {
		status = cModel.CompletionStatusAccepted
	} else {
		status = cModel.CompletionStatusRejected
	}
	err := m.state.finish(status)

	if err != nil {
		return nil, err
	}

	// try to get first unprocessed inspect
	for _, input := range m.inspects {
		if input.Status == cModel.CompletionStatusUnprocessed {
			m.state = newRollupsStateInspect(input, m.getProcessedInputCount)
			return *input, nil
		}
	}

	ctx := context.Background()

	// try to get first unprocessed advance
	input, err := m.inputRepository.FindByStatus(ctx, cModel.CompletionStatusUnprocessed)

	if err != nil {
		return nil, err
	}
	if input != nil {
		m.state = newRollupsStateAdvance(
			input,
			m.decoder,
			m.reportRepository,
			m.inputRepository,
			m.voucherRepository,
		)
		return *input, nil
	}

	// if no input was found, set state to idle
	m.state = newRollupsStateIdle()
	return nil, nil
}
