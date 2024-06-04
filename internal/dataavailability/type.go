package dataavailability

import "github.com/labstack/echo/v4"

type Fetch interface {
	Fetch(ctx echo.Context, id string) (*string, *HttpCustomError)
}
