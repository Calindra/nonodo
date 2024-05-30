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
	return ibs.model.FinishAndGetNext(accept)
}

type Sequencer interface {
	FinishAndGetNext(accept bool) Input
}
