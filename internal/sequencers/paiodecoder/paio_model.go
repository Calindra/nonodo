package paiodecoder

import "fmt"

type PaioBatch struct {
	SequencerPaymentAddress string            `json:"sequencer_payment_address"`
	Txs                     []PaioTransaction `json:"txs"`
}

type PaioTransaction struct {
	App         string        `json:"app"`
	Nonce       uint64        `json:"nonce"`
	MaxGasPrice uint64        `json:"max_gas_price"`
	Data        []byte        `json:"data"`
	Signature   PaioSignature `json:"signature"`
}

type PaioSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V string `json:"v"`
}

func (ps *PaioSignature) Hex() string {
	expectedSize := 64
	r := fmt.Sprintf("%0*s", expectedSize, ps.R[2:])
	s := fmt.Sprintf("%0*s", expectedSize, ps.S[2:])
	return fmt.Sprintf("0x%s%s%s", r, s, ps.V[2:])
}

type PaioMessage struct {
	App         string
	Nonce       string
	MaxGasPrice string
	Payload     []byte
}
