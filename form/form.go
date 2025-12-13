package form

import (
	"fmt"
	"os"

	"github.com/kkyr/fig"

	"github.com/wneessen/js-mailer/config"
)

// Form reflect the configuration struct for form configurations
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
		Turnstile struct {
			Enabled   bool   `fig:"enabled"`
			SecretKey string `fig:"secret_key"`
		}
		PrivateCaptcha struct {
			Host    string `fig:"host"`
			Enabled bool   `fig:"enabled"`
			SiteKey string `fig:"site_key"`
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

// NewForm returns a new Form object to the caller. It fails with an error when
// the form is question wasn't found or does not fulfill the syntax requirements
func NewForm(c *config.Config, i string) (Form, error) {
	root, err := os.OpenRoot(c.Forms.Path)
	if err != nil {
		return Form{}, fmt.Errorf("failed to open root of forms path: %w", err)
	}
	_, err = root.Stat(fmt.Sprintf("%s.json", i))
	if err != nil {
		return Form{}, fmt.Errorf("failed to stat form config: %w", err)
	}
	var formObj Form
	if err = fig.Load(&formObj, fig.File(fmt.Sprintf("%s.json", i)), fig.Dirs(c.Forms.Path)); err != nil {
		return Form{}, fmt.Errorf("failed to read form config: %w", err)
	}

	return formObj, nil
}
