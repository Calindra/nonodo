package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Register the health API to echo
func Register(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Ok")
	})
}
