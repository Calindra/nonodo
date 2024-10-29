package synchronizernode

import (
	"log/slog"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type AbiDecoder struct {
	abi *abi.ABI
}

func NewAbiDecoder(abi *abi.ABI) *AbiDecoder {
	return &AbiDecoder{abi: abi}
}

func (s AbiDecoder) GetMapRaw(rawData []byte) (map[string]any, error) {
	data := make(map[string]any)
	methodId := rawData[:4]
	method, err := s.abi.MethodById(methodId)
	if err != nil {
		return nil, err
	}
	err = method.Inputs.UnpackIntoMap(data, rawData[4:])
	slog.Debug("DecodedData", "map", data)
	return data, err
}
