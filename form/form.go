package form

import (
	"fmt"
	"github.com/cyphar/filepath-securejoin"
	"github.com/kkyr/fig"
	"github.com/wneessen/js-mailer/config"
	"os"
)

// Form reflect the configuration struct for form configurations
type Form struct {
	Content struct {
		Subject string
		Fields  []string
	}
	Confirmation struct {
		Enabled        bool   `fig:"enabled"`
		RecipientField string `fig:"rcpt_field" validate:"required"`
		Subject        string `fig:"subject" validate:"required"`
		Content        string `fig:"content" validate:"required"`
	}
	Domains    []string `fig:"domains" validate:"required"`
	Id         string   `fig:"id" validate:"required"`
	Recipients []string `fig:"recipients" validate:"required"`
	Secret     string   `fig:"secret" validate:"required"`
	Sender     string   `fig:"sender" validate:"required"`
	Server     struct {
		Host     string `fig:"host" validate:"required"`
		Port     int    `fig:"port" default:"25"`
		Username string
		Password string
		Timeout  string `fig:"timeout" default:"5s"`
		ForceTLS bool   `fig:"force_tls"`
	}
	Validation struct {
		Fields   []ValidationField `fig:"fields"`
		Hcaptcha struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
		Honeypot  *string `fig:"honeypot"`
		Recaptcha struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
	}
}

// ValidationField reflects the struct for a form validation field
type ValidationField struct {
	Name     string `fig:"name" validate:"required"`
	Required bool   `fig:"required"`
	Type     string `fig:"type"`
	Value    string `fig:"value"`
}

// NewForm returns a new Form object to the caller. It fails with an error when
// the form is question wasn't found or does not fulfill the syntax requirements
func NewForm(c *config.Config, i string) (Form, error) {
	formPath, err := securejoin.SecureJoin(c.Forms.Path, fmt.Sprintf("%s.json", i))
	if err != nil {
		return Form{}, fmt.Errorf("failed to securely join forms path and form id")
	}
	_, err = os.Stat(formPath)
	if err != nil {
		return Form{}, fmt.Errorf("failed to stat form config: %s", err)
	}
	var formObj Form
	if err := fig.Load(&formObj, fig.File(fmt.Sprintf("%s.json", i)),
		fig.Dirs(c.Forms.Path)); err != nil {
		return Form{}, fmt.Errorf("failed to read form config: %s", err)
	}

	return formObj, nil
}
