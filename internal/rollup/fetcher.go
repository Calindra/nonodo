package rollup

import (
	"github.com/calindra/nonodo/internal/model"
	"github.com/labstack/echo/v4"
)

type Fetch interface {
	Fetch(ctx echo.Context, id string) (*string, *model.HttpCustomError)
}
