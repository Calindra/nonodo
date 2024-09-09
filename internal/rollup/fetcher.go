package rollup

import (
	"net/http"

	DA "github.com/calindra/nonodo/internal/dataavailability"
	"github.com/ipfs/go-cid"
	"github.com/labstack/echo/v4"
	"github.com/multiformats/go-multihash"
)

var initCID string = "QmX4YVvwf6tqJfH4Gmp258ZcevAnyzifdV1XxZH3QtVwnW"

func generateCid(data []byte) (string, error) {
	// Cria o multihash do conteúdo (usando sha2-256 como exemplo)
	hash, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		return "", err
	}

	// Cria o CID usando o hash gerado
	c := cid.NewCidV1(cid.Raw, hash)
	return c.String(), nil
}

func (r *RollupAPI) Fetcher(ctx echo.Context, request GioJSONRequestBody) (*GioResponseRollup, *DA.HttpCustomError) {
	var (
		espresso             uint16 = 2222
		syscoin              uint16 = 5700
		celestia             uint16 = 714
		its_ok               uint16 = 42
		lambada_open_state   uint16 = 32
		lambada_commit_state uint16 = 33
	)

	switch request.Domain {
	case espresso:
		espressoFetcher := DA.NewEspressoFetcher(r.model.GetInputRepository())
		data, err := espressoFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case syscoin:
		syscoinFetcher := DA.NewSyscoinClient()
		data, err := syscoinFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case celestia:
		celestiaFetcher := DA.NewCelestiaClient()
		data, err := celestiaFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case lambada_open_state:

		return &GioResponseRollup{Data: "Ox" + initCID, Code: its_ok}, nil
	case lambada_commit_state:
		generatedCid, err := generateCid([]byte(request.Id))
		if err != nil {
			return nil, DA.NewHttpCustomError(http.StatusInternalServerError, nil)
		}
		return &GioResponseRollup{Data: generatedCid, Code: its_ok}, nil
	default:
		unsupported := "Unsupported domain"
		return nil, DA.NewHttpCustomError(http.StatusBadRequest, &unsupported)
	}
}
