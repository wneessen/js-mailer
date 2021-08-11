package validation

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/form"
	"net/http"
	"regexp"
)

// Field validates the form field based on its configured type
func Field(r *http.Request, f *form.ValidationField) error {
	l := log.WithFields(log.Fields{
		"action":    "validation.Field",
		"fieldName": f.Name,
	})

	if f.Required && r.Form.Get(f.Name) == "" {
		l.Debugf("Form is missing required field: %s", f.Name)
		return fmt.Errorf("field is required, but missing")
	}

	switch f.Type {
	case "text":
		return nil
	case "email":
		mailRegExp, err := regexp.Compile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
		if err != nil {
			l.Errorf("Failed to compile email comparison regexp: %s", err)
			return nil
		}
		if !mailRegExp.Match([]byte(r.Form.Get(f.Name))) {
			l.Debugf("Form field is expected to be of type email but does not match this requirementd: %s", f.Name)
			return fmt.Errorf("field is expected to be of type email, but does not match")
		}
	case "number":
		numRegExp, err := regexp.Compile("^[0-9]+$")
		if err != nil {
			l.Errorf("Failed to compile email comparison regexp: %s", err)
			return nil
		}
		if !numRegExp.Match([]byte(r.Form.Get(f.Name))) {
			l.Debugf("Form field is expected to be of type number but does not match this requirementd: %s", f.Name)
			return fmt.Errorf("field is expected to be of type number, but does not match")
		}
	case "bool":
		boolRegExp, err := regexp.Compile("^(?i)(true|false|0|1)$")
		if err != nil {
			l.Errorf("Failed to compile boolean comparison regexp: %s", err)
			return nil
		}
		if !boolRegExp.Match([]byte(r.Form.Get(f.Name))) {
			l.Debugf("Form field is expected to be of type boolean but does not match this requirementd: %s", f.Name)
			return fmt.Errorf("field is expected to be of type bool, but does not match")
		}
	default:
		return nil
	}

	return nil
}
