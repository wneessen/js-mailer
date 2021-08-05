package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ErrorResponseJson struct {
	StatusCode   int    `json:"status_code"`
	ErrorMessage string `json:"error_message"`
}

type MissingFieldsResponseJson struct {
	StatusCode    int      `json:"status_code"`
	ErrorMessage  string   `json:"error_message"`
	MissingFields []string `json:"missing_fields"`
}

func ErrorJson(w http.ResponseWriter, c int, m string) {
	l := log.WithFields(log.Fields{
		"action": "http_error.ErrorJson",
	})
	l.Debugf("Request failed with code %d: %s", c, m)
	errorMsg := ErrorResponseJson{StatusCode: c, ErrorMessage: m}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(errorMsg); err != nil {
		l.Errorf("Failed to write error response JSON: %s", err)
	}
}

func MissingFieldsJson(w http.ResponseWriter, c int, m string, f []string) {
	l := log.WithFields(log.Fields{
		"action": "http_error.MissingFieldsJson",
	})
	l.Debugf("Request failed with code %d: %s", c, m)
	errorMsg := MissingFieldsResponseJson{
		StatusCode:    c,
		ErrorMessage:  m,
		MissingFields: f,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(errorMsg); err != nil {
		l.Errorf("Failed to write error response JSON: %s", err)
	}
}
