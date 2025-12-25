// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package forms

import (
	"errors"
	"fmt"
	"os"

	"github.com/kkyr/fig"
)

var ErrFormNotFound = errors.New("form not found")

// Form is the configuration struct for a form
type Form struct {
	Content struct {
		Subject string
		Fields  []string
	}
	Confirmation struct {
		Enabled        bool   `fig:"enabled"`
		RecipientField string `fig:"rcpt_field"`
		Subject        string `fig:"subject"`
		Content        string `fig:"content"`
	}
	Domains    []string `fig:"domains" validate:"required"`
	AttachCSV  bool     `fig:"attach_csv"`
	ID         string   `fig:"id" validate:"required"`
	Recipients []string `fig:"recipients" validate:"required"`
	ReplyTo    struct {
		Field string `json:"field"`
	}
	Secret string `fig:"secret" validate:"required"`
	Sender string `fig:"sender" validate:"required"`
	Server struct {
		Host     string `fig:"host" validate:"required"`
		Port     int    `fig:"port" default:"25"`
		Username string
		Password string
		ForceTLS bool `fig:"force_tls"`
		DryRun   bool `fig:"dry_run"`
	}
	Validation struct {
		DisableSubmissionSpeedCheck bool              `fig:"disable_submission_speed_check"`
		RandomAntiSpamField         bool              `fig:"random_anti_spam_field"`
		Fields                      []ValidationField `fig:"fields"`
		Hcaptcha                    struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
		Honeypot  string `fig:"honeypot"`
		Recaptcha struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
		Turnstile struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
		PrivateCaptcha struct {
			Host    string `fig:"host"`
			Enabled bool   `fig:"enabled"`
			APIKey  string `fig:"api_key"`
		} `fig:"private_captcha"`
	}
}

// ValidationField reflects the struct for a form validation field
type ValidationField struct {
	Name     string `fig:"name" validate:"required"`
	Required bool   `fig:"required"`
	Type     string `fig:"type"`
	Value    string `fig:"value"`
}

func New(path, formID string) (*Form, error) {
	form := new(Form)

	root, err := os.OpenRoot(path)
	if err != nil {
		return form, fmt.Errorf("failed to open root of form path: %w", err)
	}

	var formFile string
	exts := []string{"toml", "yaml", "yml", "json"}
	for _, ext := range exts {
		if _, err = root.Stat(formID + "." + ext); err == nil {
			formFile = formID + "." + ext
			break
		}
	}
	if formFile == "" {
		return form, ErrFormNotFound
	}

	if err = fig.Load(form, fig.File(formFile), fig.Dirs(root.Name())); err != nil {
		return form, fmt.Errorf("failed parse form config: %w", err)
	}

	return form, nil
}
