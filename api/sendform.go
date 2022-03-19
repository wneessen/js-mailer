package api

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/wneessen/go-mail"
	"github.com/wneessen/js-mailer/form"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"time"
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
	var mailerr error
	mailMsg := mail.NewMsg()
	mailerr = mailMsg.From(sr.FormObj.Sender)
	mailerr = mailMsg.To(sr.FormObj.Recipients...)
	mailMsg.Subject(sr.FormObj.Content.Subject)
	if sr.FormObj.ReplyTo.Field != "" {
		sf := c.FormValue(sr.FormObj.ReplyTo.Field)
		if sf != "" {
			mailerr = mailMsg.ReplyTo(sf)
		}
	}
	if mailerr != nil {
		c.Logger().Errorf("failed to generate mail object: %s", mailerr)
		return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
			Message: "could not connect to configured mail server",
			Data:    mailerr.Error(),
		})
	}

	mailBody := "The following form fields have been transmitted:\n"
	for _, k := range sr.FormObj.Content.Fields {
		if v := c.FormValue(k); v != "" {
			mailBody = fmt.Sprintf("%s\n* %s => %s", mailBody, k, v)
		}
	}
	mailMsg.SetBodyString(mail.TypeTextPlain, mailBody)

	// Send the mail message
	mc, err := GetMailClient(sr.FormObj)
	if err != nil {
		c.Logger().Errorf("Could not create new mail client: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
			Message: "Cloud not create new mail client",
			Data:    err.Error(),
		})
	}
	if err := mc.DialAndSend(mailMsg); err != nil {
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
	var mailerr error
	mailMsg := mail.NewMsg()
	mailerr = mailMsg.From(f.Sender)
	mailerr = mailMsg.To(r)
	mailMsg.Subject(f.Confirmation.Subject)
	mailMsg.SetBodyString(mail.TypeTextPlain, f.Confirmation.Content)
	if mailerr != nil {
		return fmt.Errorf("failed to create mail message: %w", mailerr)
	}

	mc, err := GetMailClient(f)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	if err := mc.DialAndSend(mailMsg); err != nil {
		return fmt.Errorf("could not send confirmation mail message: %w", err)
	}
	return nil
}

// GetMailClient returns a new mail dailer object based on the form configuration
func GetMailClient(f *form.Form) (*mail.Client, error) {
	var serverTimeout time.Duration
	serverTimeout, err := time.ParseDuration(f.Server.Timeout)
	if err != nil {
		serverTimeout = time.Second * 5
	}
	mc, err := mail.NewClient(f.Server.Host, mail.WithPort(f.Server.Port),
		mail.WithUsername(f.Server.Username), mail.WithPassword(f.Server.Password),
		mail.WithSMTPAuth(mail.SMTPAuthPlain), mail.WithTimeout(serverTimeout))

	if err != nil {
		return mc, err
	}
	if !f.Server.ForceTLS {
		mc.SetTLSPolicy(mail.TLSOpportunistic)
	}
	return mc, nil
}
