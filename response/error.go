package response

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorObj is the structure of an error
type ErrorObj struct {
	Message string
	Data    interface{}
}

// ErrorResponse reflects the default JSON structure of an error response
type ErrorResponse struct {
	StatusCode   int         `json:"status_code"`
	Status       string      `json:"status"`
	ErrorMessage string      `json:"error_message"`
	ErrorData    interface{} `json:"error_data,omitempty"`
}

// CustomError is a custom error handler for the echo.NewHTTPError function
func CustomError(err error, c echo.Context) {
	errResp := &ErrorResponse{
		StatusCode: http.StatusInternalServerError,
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		errResp.StatusCode = he.Code
		errResp.Status = http.StatusText(errResp.StatusCode)
		if em, ok := he.Message.(*echo.HTTPError); ok {
			errResp.ErrorMessage = em.Message.(string)
		}
		if em, ok := he.Message.(*ErrorObj); ok {
			errResp.ErrorMessage = em.Message
			errResp.ErrorData = em.Data
		}
		if em, ok := he.Message.(string); ok {
			errResp.ErrorMessage = em
		}
	}

	if err := c.JSON(errResp.StatusCode, errResp); err != nil {
		c.Logger().Errorf("Failed to render error JSON: %s", err)
	}
}
