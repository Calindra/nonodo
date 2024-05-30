package model

type InputBoxSequencer struct {
	model *NonodoModel
}

func NewInputBoxSequencer(model *NonodoModel) *InputBoxSequencer {
	return &InputBoxSequencer{model: model}
}

type EspressoSequencer struct {
	//??
}

func (es *EspressoSequencer) FinishAndGetNext(accept bool) Input {
	return nil
}

func (ibs *InputBoxSequencer) FinishAndGetNext(accept bool) Input {
	m := ibs.model

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// finish current input
	var status CompletionStatus
	if accept {
		status = CompletionStatusAccepted
	} else {
		status = CompletionStatusRejected
	}
	m.state.finish(status)

	// try to get first unprocessed inspect
	for _, input := range m.inspects {
		if input.Status == CompletionStatusUnprocessed {
			m.state = newRollupsStateInspect(input, m.getProcessedInputCount)
			return *input
		}
	}

	// try to get first unprocessed advance
	input, err := m.inputRepository.FindByStatus(CompletionStatusUnprocessed)
	if err != nil {
		panic(err)
	}
	if input != nil {
		m.state = newRollupsStateAdvance(
			input,
			m.decoder,
			m.reportRepository,
			m.inputRepository,
		)
		return *input
	}

	// if no input was found, set state to idle
	m.state = newRollupsStateIdle()
	return nil
}

type Sequencer interface {
	FinishAndGetNext(accept bool) Input
}
