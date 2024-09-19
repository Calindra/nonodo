package dataavailability

import "github.com/labstack/echo/v4"

type AvailFetcher struct{}

// Fetch implements Fetch.
func (a *AvailFetcher) Fetch(ctx echo.Context, id string) (*string, *HttpCustomError) {
	panic("unimplemented")
}

func NewAvailFetcher() Fetch {
	return &AvailFetcher{}
}
