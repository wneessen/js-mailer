package http_error

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type ErrorResponseJson struct {
	StatusCode   int
	ErrorMessage string
}

func ErrorJson(w http.ResponseWriter, c int, m string) {
	l := log.WithFields(log.Fields{
		"action": "main.errorResponse",
	})
	l.Debugf("Request failed with code %d: %s", c, m)
	errorMsg := ErrorResponseJson{StatusCode: c, ErrorMessage: m}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(errorMsg); err != nil {
		l.Errorf("Failed to write error response JSON: %s", err)
	}
}
