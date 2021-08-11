package form

import (
	"fmt"
	"github.com/kkyr/fig"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/config"
	"os"
)

// Form reflect the configuration struct for form configurations
type Form struct {
	Id         string   `fig:"id" validate:"required"`
	Secret     string   `fig:"secret" validate:"required"`
	Recipients []string `fig:"recipients" validate:"required"`
	Sender     string   `fig:"sender" validate:"required"`
	Domains    []string `fig:"domains" validate:"required"`
	Hcaptcha   struct {
		Enabled   bool   `fig:"enabled"`
		SecretKey string `fig:"secret_key"`
	}
	Content struct {
		Subject        string
		Fields         []string
		RequiredFields []string `fig:"required_fields"`
		Honeypot       *string  `fig:"honeypot"`
	}
	Server struct {
		Host     string `fig:"host" validate:"required"`
		Port     int    `fig:"port" default:"25"`
		Username string
		Password string
		Timeout  string `fig:"timeout" default:"5s"`
		ForceTLS bool   `fig:"force_tls"`
	}
}

// NewForm returns a new Form object to the caller. It fails with an error when
// the form is question wasn't found or does not fulfill the syntax requirements
func NewForm(c *config.Config, i string) (Form, error) {
	l := log.WithFields(log.Fields{
		"action": "form.NewForm",
	})
	_, err := os.Stat(fmt.Sprintf("%s/%s.json", c.Forms.Path, i))
	if err != nil {
		l.Errorf("Failed to stat form config: %s", err)
		return Form{}, fmt.Errorf("Not a valid form id")
	}
	var formObj Form
	if err := fig.Load(&formObj, fig.File(fmt.Sprintf("%s.json", i)),
		fig.Dirs(c.Forms.Path)); err != nil {
		l.Errorf("Failed to read form config: %s", err)
		return Form{}, fmt.Errorf("Not a valid form id")
	}

	return formObj, nil
}
