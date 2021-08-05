package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/form"
	"net/http"
)

type SuccessResponseJson struct {
	StatusCode     int    `json:"status_code"`
	SuccessMessage string `json:"success_message"`
	FormId         int
}

func SuccessJson(w http.ResponseWriter, c int, f *form.Form) {
	l := log.WithFields(log.Fields{
		"action": "http_error.ErrorJson",
	})
	l.Debug("Request successfully completed")
	successMsg := SuccessResponseJson{
		StatusCode:     c,
		SuccessMessage: "Message successfully sent",
		FormId:         f.Id,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(successMsg); err != nil {
		l.Errorf("Failed to write success response JSON: %s", err)
	}
}
