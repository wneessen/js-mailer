// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/wneessen/go-mail"

	"github.com/wneessen/js-mailer/internal/forms"
)

var (
	// version is the version of the application (will be set at build time)
	version = "dev"

	// userAgent is the User-Agent that the HTTP client sends with API requests
	userAgent = fmt.Sprintf("js-mailer/%s // https://github.com/wneessen/js-mailer", version)
)

func (s *Server) sendMail(r *http.Request, form *forms.Form) (string, string, error) {
	if form.Server.DryRun {
		s.log.Info("dry-run mode enabled, skipping actual mail delivery")
		return "dry-run succeeded", "dry-run succeeded", nil
	}

	var confirmationResponse, messageResponse string

	// Initialize mail client
	client, err := mail.NewClient(form.Server.Host, mail.WithPort(form.Server.Port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithUsername(form.Server.Username),
		mail.WithPassword(form.Server.Password), mail.WithTLSPolicy(mail.DefaultTLSPolicy),
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create mail client: %w", err)
	}
	if !form.Server.ForceTLS {
		client.SetTLSPolicy(mail.TLSOpportunistic)
	}

	// Send confirmation mail
	if form.Confirmation.Enabled {
		confirmationResponse, err = s.sendConfirmation(r, form, client)
		if err != nil {
			return "", "", fmt.Errorf("failed to send confirmation mail: %w", err)
		}
	}

	// Send actual message
	messageResponse, err = s.sendMessage(r, form, client)
	if err != nil {
		return "", "", fmt.Errorf("failed to send message: %w", err)
	}

	return confirmationResponse, messageResponse, nil
}

func (s *Server) sendConfirmation(r *http.Request, form *forms.Form, client *mail.Client) (string, error) {
	rcpt := r.FormValue(form.Confirmation.RecipientField)
	if rcpt == "" {
		return "", fmt.Errorf("confirmation mail feature activated, but recipient field is empty")
	}

	message := mail.NewMsg()
	if err := message.From(form.Sender); err != nil {
		return "", fmt.Errorf("failed to set sender address: %w", err)
	}
	if err := message.To(rcpt); err != nil {
		return "", fmt.Errorf("failed to set recipient address: %w", err)
	}
	message.Subject(form.Confirmation.Subject)
	message.SetBodyString(mail.TypeTextPlain, form.Confirmation.Content)
	message.SetUserAgent(userAgent)

	if err := client.DialAndSendWithContext(r.Context(), message); err != nil {
		return "", fmt.Errorf("failed to send confirmation mail: %w", err)
	}

	return message.ServerResponse(), nil
}

func (s *Server) sendMessage(r *http.Request, form *forms.Form, client *mail.Client) (string, error) {
	message := mail.NewMsg()
	if err := message.From(form.Sender); err != nil {
		return "", fmt.Errorf("failed to set sender address: %w", err)
	}
	if err := message.To(form.Recipients...); err != nil {
		return "", fmt.Errorf("failed to set recipient address: %w", err)
	}
	message.Subject(form.Content.Subject)
	message.SetUserAgent(userAgent)

	if form.ReplyTo.Field != "" {
		replyto := r.FormValue(form.ReplyTo.Field)
		if replyto == "" {
			return "", fmt.Errorf("reply-to field is set, but no value was provided")
		}
		if err := message.ReplyTo(replyto); err != nil {
			return "", fmt.Errorf("failed to set reply-to address: %w", err)
		}
	}

	body := strings.Builder{}
	body.WriteString("The following form fields have been transmitted:\n\n")
	for _, field := range form.Content.Fields {
		if val := r.FormValue(field); val != "" {
			body.WriteString(fmt.Sprintf("* %s => %s\n", field, val))
		}
	}
	message.SetBodyString(mail.TypeTextPlain, body.String())

	if err := client.DialAndSendWithContext(r.Context(), message); err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return message.ServerResponse(), nil
}
