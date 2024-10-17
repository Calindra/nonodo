package paio

import (
	"fmt"
	"log/slog"

	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/sequencers/avail"
)

type PaioBuilder struct {
	AvalClient      *avail.AvailClient
	InputRepository *repository.InputRepository
	RpcUrl          string
	EspressoUrl     string
	PaioServerUrl   string
}

func NewPaioBuilder() *PaioBuilder {
	return &PaioBuilder{
		AvalClient:      nil,
		InputRepository: nil,
		RpcUrl:          "",
		EspressoUrl:     "",
		PaioServerUrl:   "",
	}
}

func (pb *PaioBuilder) WithAvalClient(availClient *avail.AvailClient) *PaioBuilder {
	pb.AvalClient = availClient
	return pb
}

func (pb *PaioBuilder) WithInputRepository(inputRepository *repository.InputRepository) *PaioBuilder {
	pb.InputRepository = inputRepository
	return pb
}

func (pb *PaioBuilder) WithRpcUrl(rpcUrl string) *PaioBuilder {
	pb.RpcUrl = rpcUrl
	return pb
}

func (pb *PaioBuilder) WithEspressoUrl(espressoUrl string) *PaioBuilder {
	pb.EspressoUrl = espressoUrl
	return pb
}

func (pb *PaioBuilder) WithPaioServerUrl(paioServerUrl string) *PaioBuilder {
	pb.PaioServerUrl = paioServerUrl
	return pb
}

func (pb *PaioBuilder) Build() *PaioAPI {
	var clientSender Sender

	if pb.EspressoUrl != "" {
		clientSender = NewEspressoSender(pb.EspressoUrl)
	}

	paioNonceUrl := ""
	if pb.PaioServerUrl != "" {
		slog.Info("Using Paio's server", "url", pb.PaioServerUrl)
		clientSender = NewPaioSender2Server(pb.PaioServerUrl)
		paioNonceUrl = fmt.Sprintf("%s/nonce", pb.PaioServerUrl)
	}

	return &PaioAPI{
		availClient:     pb.AvalClient,
		inputRepository: pb.InputRepository,
		EvmRpcUrl:       pb.RpcUrl,
		ClientSender:    clientSender,
		chainID:         nil,
		paioNonceUrl:    paioNonceUrl,
	}
}