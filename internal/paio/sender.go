package paio

import (
	"github.com/calindra/nonodo/internal/sequencers/espresso"
	"github.com/cartesi/rollups-graphql/pkg/commons"
)

type Sender interface {
	SubmitSigAndData(sigAndData commons.SigAndData) (string, error)
}

type EspressoSender struct {
	Namespace uint64
	Client    espresso.EspressoClient
}

// SubmitSigAndData implements Sender.
func (es EspressoSender) SubmitSigAndData(sigAndData commons.SigAndData) (string, error) {
	return es.Client.SubmitSigAndData(int(es.Namespace), sigAndData)
}

func NewEspressoSender(url string, namespace uint64) Sender {
	return EspressoSender{
		Namespace: namespace,
		Client: espresso.EspressoClient{
			EspressoUrl: url,
		},
	}
}
