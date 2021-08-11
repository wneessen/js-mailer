package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// SuccessResponseJson reflects the HTTP response JSON for a successful request
type SuccessResponseJson struct {
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Data       interface{} `json:"data"`
}

// SuccessJson writes a SuccessResponseJson struct to the http.ResponseWriter
func SuccessJson(w http.ResponseWriter, c int, d interface{}) {
	l := log.WithFields(log.Fields{
		"action": "http_error.ErrorJson",
	})
	l.Debug("Request successfully completed")
	successMsg := SuccessResponseJson{
		StatusCode: c,
		Status:     HttpStatusMsg[c],
		Data:       d,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(successMsg); err != nil {
		l.Errorf("Failed to write success response JSON: %s", err)
	}
}
