package rollup

import (
	"net/http"

	"github.com/calindra/nonodo/internal/model"
	"github.com/labstack/echo/v4"
)

type Fetch interface {
	Fetch(ctx echo.Context, id string) (*string, *model.HttpCustomError)
}

func (r *RollupAPI) Fetcher(ctx echo.Context, request GioJSONRequestBody) (*GioResponseRollup, *model.HttpCustomError) {
	var (
		espresso uint16 = 2222
		syscoin  uint16 = 5700
		its_ok   uint16 = 42
	)

	switch request.Domain {
	case espresso:
		espressoFetcher := r.NewEspressoFetcher(r.model.GetInputRepository())
		data, err := espressoFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case syscoin:
		syscoinFetcher := r.NewSyscoinClient()
		data, err := syscoinFetcher.Fetch(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	default:
		unsupported := "Unsupported domain"
		return nil, model.NewHttpCustomError(http.StatusBadRequest, &unsupported)
	}
}
