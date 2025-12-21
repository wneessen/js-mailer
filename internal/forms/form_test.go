// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package forms

import (
	"slices"
	"testing"
)

const (
	testFormID                    = "contact-form"
	testFormSecret                = "test-secret-key"
	testFormSender                = "no-reply@example.com"
	testFormContentSubject        = "Contact form submission"
	testFormConfirmationEnabled   = true
	testFormConfirmationSubject   = "We received your message"
	testFormConfirmationContent   = "Thank you for contacting us. We will get back to you shortly."
	testFormConfirmationRcptField = "email"
	testFormReplyToField          = "email"
	testFormServerHost            = "smtp.example.com"
	testFormServerPort            = 587
	testFormServerUsername        = "smtp-user"
	testFormServerPassword        = "smtp-password"
	testFormServerForceTLS        = true
)

var (
	testFormContentFields = []string{"name", "email", "message"}
	testFormDomains       = []string{"example.com", "www.example.com"}
	testFormRecipients    = []string{
		"support@example.com",
		"sales@example.com",
	}
	testFormValidationFields = []ValidationField{
		{
			Name:     "email",
			Required: true,
			Type:     "email",
			Value:    "",
		},
		{
			Name:     "message",
			Required: true,
			Type:     "string",
			Value:    "",
		},
	}
)

func TestNew(t *testing.T) {
	t.Run("read forms from file in different formats", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			file     string
			succeeds bool
		}{
			{"json", "../../testdata", "testform_json", true},
			{"yaml", "../../testdata", "testform_yaml", true},
			{"toml", "../../testdata", "testform_toml", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config, err := New(tt.path, tt.file)
				if tt.succeeds && err != nil {
					t.Fatalf("failed to read form from file: %s", err)
				}
				if config.ID != testFormID {
					t.Errorf("expected form ID to be %s, got %s", testFormID, config.ID)
				}
				if config.Secret != testFormSecret {
					t.Errorf("expected form secret to be %s, got %s", testFormSecret, config.Secret)
				}
				if config.Sender != testFormSender {
					t.Errorf("expected form sender to be %s, got %s", testFormSender, config.Sender)
				}
				if config.Content.Subject != testFormContentSubject {
					t.Errorf("expected form content subject to be %s, got %s", testFormContentSubject,
						config.Content.Subject)
				}
				if slices.Compare(config.Content.Fields, testFormContentFields) != 0 {
					t.Errorf("expected form content fields to be %s, got %s", testFormContentFields,
						config.Content.Fields)
				}
				if config.Server.Host != testFormServerHost {
					t.Errorf("expected form server host to be %s, got %s", testFormServerHost,
						config.Server.Host)
				}
				if config.Server.Port != testFormServerPort {
					t.Errorf("expected form server port to be %d, got %d", testFormServerPort,
						config.Server.Port)
				}
				if config.Server.Username != testFormServerUsername {
					t.Errorf("expected form server username to be %s, got %s", testFormServerUsername,
						config.Server.Username)
				}
				if config.Server.Password != testFormServerPassword {
					t.Errorf("expected form server password to be %s, got %s", testFormServerPassword,
						config.Server.Password)
				}
				if config.Server.ForceTLS != testFormServerForceTLS {
					t.Errorf("expected form server force TLS to be %t, got %t", testFormServerForceTLS,
						config.Server.ForceTLS)
				}
				if config.Confirmation.Enabled != testFormConfirmationEnabled {
					t.Errorf("expected form confirmation to be %t, got %t", testFormConfirmationEnabled,
						config.Confirmation.Enabled)
				}
				if config.Confirmation.Subject != testFormConfirmationSubject {
					t.Errorf("expected form confirmation subject to be %s, got %s", testFormConfirmationSubject,
						config.Confirmation.Subject)
				}
				if config.Confirmation.Content != testFormConfirmationContent {
					t.Errorf("expected form confirmation content to be %s, got %s", testFormConfirmationContent,
						config.Confirmation.Content)
				}
				if config.Confirmation.RecipientField != testFormConfirmationRcptField {
					t.Errorf("expected form confirmation recipient field to be %s, got %s", testFormConfirmationRcptField,
						config.Confirmation.RecipientField)
				}
				if config.ReplyTo.Field != testFormReplyToField {
					t.Errorf("expected form reply-to field to be %s, got %s", testFormReplyToField,
						config.ReplyTo.Field)
				}
				if slices.Compare(config.Domains, testFormDomains) != 0 {
					t.Errorf("expected form domains to be %s, got %s", testFormDomains, config.Domains)
				}
				if slices.Compare(config.Recipients, testFormRecipients) != 0 {
					t.Errorf("expected form recipients to be %s, got %s", testFormRecipients, config.Recipients)
				}
				if len(config.Validation.Fields) != len(testFormValidationFields) {
					t.Errorf("expected form validation fields to be %d, got %d", len(testFormValidationFields),
						len(config.Validation.Fields))
				}
				for i, field := range config.Validation.Fields {
					if field.Name != testFormValidationFields[i].Name {
						t.Errorf("expected form validation field %d to be %s, got %s", i,
							testFormValidationFields[i].Name, field.Name)
					}
					if field.Type != testFormValidationFields[i].Type {
						t.Errorf("expected form validation field %d to be %s, got %s", i,
							testFormValidationFields[i].Type, field.Type)
					}
					if field.Required != testFormValidationFields[i].Required {
						t.Errorf("expected form validation field %d to be %t, got %t", i,
							testFormValidationFields[i].Required, field.Required)
					}
					if field.Value != testFormValidationFields[i].Value {
						t.Errorf("expected form validation field %d to be %s, got %s", i,
							testFormValidationFields[i].Value, field.Value)
					}
				}
			})
		}
	})
	t.Run("reading form fails due to invalid path", func(t *testing.T) {
		_, err := New("../../testdata-non-existing", "testform_toml")
		if err == nil {
			t.Fatal("expected error when reading form from invalid path")
		}
	})
	t.Run("reading form fails due to non-existing file", func(t *testing.T) {
		_, err := New("../../testdata", "non_existing_file")
		if err == nil {
			t.Fatal("expected error when reading form from non-existing file")
		}
	})
	t.Run("reading form fails due to config being incomplete", func(t *testing.T) {
		_, err := New("../../testdata", "incomplete_form")
		if err == nil {
			t.Fatal("expected error when reading incomplete form")
		}
	})
}
