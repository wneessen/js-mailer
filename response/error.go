package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// ErrorResponseJson reflects the JSON response for a failed request
type ErrorResponseJson struct {
	StatusCode   int    `json:"status_code"`
	ErrorMessage string `json:"error_message"`
}

// MissingFieldsResponseJson reflects the JSON response for a failed request due to
// missing fields in the send request
type MissingFieldsResponseJson struct {
	StatusCode    int      `json:"status_code"`
	ErrorMessage  string   `json:"error_message"`
	MissingFields []string `json:"missing_fields"`
}

// ErrorJson writes a ErrorResponseJson to the http.ResponseWriter in case
// an error response is needed as result to the HTTP request
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

// MissingFieldsJson writes a MissingFieldsResponseJson to the http.ResponseWriter when
// a send request is missing required fields
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
