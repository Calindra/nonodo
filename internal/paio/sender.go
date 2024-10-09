package paio

import (
	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/sequencers/espresso"
)

type Sender interface {
	SubmitSigAndData(sigAndData commons.SigAndData) (string, error)
}

const DEFAULT_NAMESPACE = 10008

type EspressoSender struct {
	Namespace int
	Client    espresso.EspressoClient
}

// SubmitSigAndData implements Sender.
func (es EspressoSender) SubmitSigAndData(sigAndData commons.SigAndData) (string, error) {
	return es.Client.SubmitSigAndData(es.Namespace, sigAndData)
}

func NewEspressoSender(url string) Sender {
	return EspressoSender{
		Namespace: DEFAULT_NAMESPACE,
		Client: espresso.EspressoClient{
			EspressoUrl: url,
		},
	}
}
