package rollup

import (
	"github.com/calindra/nonodo/internal/model"
)

type InputBoxSequencer struct {
	model *model.NonodoModel
}

type EspressoSequencer struct {
	//??
}

func (es *EspressoSequencer) FinishAndGetNext(accept bool) model.Input {
	return nil
}

func (ibs *InputBoxSequencer) FinishAndGetNext(accept bool) model.Input {
	return ibs.model.FinishAndGetNext(accept)
}

type Sequencer interface {
	FinishAndGetNext(accept bool) model.Input
}
