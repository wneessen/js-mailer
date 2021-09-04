package api

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

// PingResponse reflects the JSON structure for a ping response
type PingResponse struct {
	StatusCode int
	Status     string
	Data       interface{}
}

// Ping is a test route for the API
func (r *Route) Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, &PingResponse{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Data: map[string]string{
			"Ping": "Pong",
		},
	})
}
