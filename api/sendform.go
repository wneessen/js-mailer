package api

import (
	"fmt"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"time"

	"github.com/go-mail/mail"
	"github.com/labstack/echo/v4"
)

// SentSuccessfull represents confirmation JSON structure for a successfully sent message
type SentSuccessfull struct {
	FormId   string `json:"form_id"`
	SendTime int64  `json:"send_time"`
}

// SendForm handles the HTTP form sending API request
func (r *Route) SendForm(c echo.Context) error {
	sr := c.Get("formobj").(*SendFormRequest)
	if sr == nil {
		c.Logger().Errorf("no form object found in context")
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
	}

	// Compose the mail message
	mailMsg := mail.NewMessage()
	mailMsg.SetHeader("From", sr.FormObj.Sender)
	mailMsg.SetHeader("To", sr.FormObj.Recipients...)
	mailMsg.SetHeader("Subject", sr.FormObj.Content.Subject)

	mailBody := "The following form fields have been transmitted:\n"
	for _, k := range sr.FormObj.Content.Fields {
		if v := c.FormValue(k); v != "" {
			mailBody = fmt.Sprintf("%s\n* %s => %s", mailBody, k, v)
		}
	}
	mailMsg.SetBody("text/plain", mailBody)

	// Send the mail message
	var serverTimeout time.Duration
	var err error
	serverTimeout, err = time.ParseDuration(sr.FormObj.Server.Timeout)
	if err != nil {
		c.Logger().Warnf("Could not parse configured server timeout: %s", err)
		serverTimeout = time.Second * 5
	}
	mailDailer := mail.NewDialer(sr.FormObj.Server.Host, sr.FormObj.Server.Port, sr.FormObj.Server.Username,
		sr.FormObj.Server.Password)
	mailDailer.Timeout = serverTimeout
	if sr.FormObj.Server.ForceTLS {
		mailDailer.StartTLSPolicy = mail.MandatoryStartTLS
	}
	mailSender, err := mailDailer.Dial()
	if err != nil {
		c.Logger().Errorf("Could not connect to configured mail server: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
			Message: "could not connect to configured mail server",
			Data:    err.Error(),
		})
	}
	defer func() {
		if err := mailSender.Close(); err != nil {
			c.Logger().Errorf("Failed to close mail server connection: %s", err)
		}
	}()
	if err := mail.Send(mailSender, mailMsg); err != nil {
		c.Logger().Errorf("Could not send mail message: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
			Message: "could not send mail message",
			Data:    err.Error(),
		})
	}

	return c.JSON(http.StatusOK, response.SuccessResponse{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Data: &SentSuccessfull{
			FormId:   sr.FormObj.Id,
			SendTime: time.Now().Unix(),
		},
	})
}
