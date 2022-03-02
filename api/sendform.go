package api

import (
	"fmt"
	"github.com/wneessen/js-mailer/form"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"time"

	"github.com/go-mail/mail"
	"github.com/labstack/echo/v4"
)

// SentSuccessful represents confirmation JSON structure for a successfully sent message
type SentSuccessful struct {
	FormId           string `json:"form_id"`
	SendTime         int64  `json:"send_time"`
	ConfirmationSent bool   `json:"confirmation_sent"`
	ConfirmationRcpt string `json:"confirmation_rcpt"`
}

// SendForm handles the HTTP form sending API request
func (r *Route) SendForm(c echo.Context) error {
	sr := c.Get("formobj").(*SendFormRequest)
	if sr == nil {
		c.Logger().Errorf("no form object found in context")
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
	}

	// Do we have some confirmation mail to handle?
	confirmWasSent := false
	confirmRcpt := ""
	if sr.FormObj.Confirmation.Enabled {
		sendConfirm := true
		confirmRcpt = c.FormValue(sr.FormObj.Confirmation.RecipientField)
		if confirmRcpt == "" {
			c.Logger().Warnf("confirmation mail feature activated, but recpienent field not found or empty")
			sendConfirm = false
		}
		if sr.FormObj.Confirmation.Subject == "" {
			c.Logger().Warnf("confirmation mail feature activated, but no subject found in configuration")
			sendConfirm = false
		}
		if sr.FormObj.Confirmation.Content == "" {
			c.Logger().Warnf("confirmation mail feature activated, but no content found in configuration")
			sendConfirm = false
		}
		if sendConfirm {
			confirmWasSent = true
			if err := SendFormConfirmation(sr.FormObj, confirmRcpt); err != nil {
				c.Logger().Warnf("failed to send confirmation mail: %s", err)
				confirmWasSent = false
			}
		}
	}

	// Compose the mail message
	mailMsg := mail.NewMessage()
	mailMsg.SetHeader("From", sr.FormObj.Sender)
	mailMsg.SetHeader("To", sr.FormObj.Recipients...)
	mailMsg.SetHeader("Subject", sr.FormObj.Content.Subject)
	if sr.FormObj.ReplyTo.Field != "" {
		sf := c.FormValue(sr.FormObj.ReplyTo.Field)
		if sf != "" {
			mailMsg.SetHeader("Reply-To", sf)
		}
	}

	mailBody := "The following form fields have been transmitted:\n"
	for _, k := range sr.FormObj.Content.Fields {
		if v := c.FormValue(k); v != "" {
			mailBody = fmt.Sprintf("%s\n* %s => %s", mailBody, k, v)
		}
	}
	mailMsg.SetBody("text/plain", mailBody)

	// Send the mail message
	mailDailer := GetMailDailer(sr.FormObj)
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
		Data: SentSuccessful{
			FormId:           sr.FormObj.Id,
			SendTime:         time.Now().Unix(),
			ConfirmationSent: confirmWasSent,
			ConfirmationRcpt: confirmRcpt,
		},
	})
}

// SendFormConfirmation sends out a confirmation mail if requested in the form
func SendFormConfirmation(f *form.Form, r string) error {
	mailMsg := mail.NewMessage()
	mailMsg.SetHeader("From", f.Sender)
	mailMsg.SetHeader("To", r)
	mailMsg.SetHeader("Subject", f.Confirmation.Subject)
	mailMsg.SetBody("text/plain", f.Confirmation.Content)
	mailDailer := GetMailDailer(f)
	mailSender, err := mailDailer.Dial()
	if err != nil {
		return fmt.Errorf("could not connect to configured mail server: %w", err)
	}
	if err := mail.Send(mailSender, mailMsg); err != nil {
		return fmt.Errorf("could not send confirmation mail message: %w", err)
	}
	if err := mailSender.Close(); err != nil {
		return fmt.Errorf("failed to close mail server connection: %w", err)
	}
	return nil
}

// GetMailDailer returns a new mail dailer object based on the form configuration
func GetMailDailer(f *form.Form) *mail.Dialer {
	var serverTimeout time.Duration
	serverTimeout, err := time.ParseDuration(f.Server.Timeout)
	if err != nil {
		serverTimeout = time.Second * 5
	}
	mailDailer := mail.NewDialer(f.Server.Host, f.Server.Port, f.Server.Username, f.Server.Password)
	mailDailer.Timeout = serverTimeout
	if f.Server.ForceTLS {
		mailDailer.StartTLSPolicy = mail.MandatoryStartTLS
	}
	return mailDailer
}
