package response

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// HttpStatusMsg is a mapping of HTTP status codes to their corresponding status message
var HttpStatusMsg = map[int]string{
	200: "Ok",
	400: "Bad Request",
	401: "Unauthorized",
	404: "Not Found",
	500: "Internal Server Error",
}

// ErrorResponseJson reflects the JSON response for a failed request
type ErrorResponseJson struct {
	StatusCode   int         `json:"status_code"`
	Status       string      `json:"status"`
	ErrorMessage string      `json:"error_message"`
	ErrorData    interface{} `json:"error_data,omitempty"`
}

// ErrorJson writes a ErrorResponseJson with no ErrorData to the http.ResponseWriter in case
// an error response is needed as result to the HTTP request
func ErrorJson(w http.ResponseWriter, c int, m string) {
	errorJson(w, c, m, nil)
}

// ErrorJsonData writes a ErrorResponseJson with ErrorData to the http.ResponseWriter in case
// an error response is needed as result to the HTTP request
func ErrorJsonData(w http.ResponseWriter, c int, m string, d interface{}) {
	errorJson(w, c, m, d)
}

// errorJson writes a ErrorResponseJson to the http.ResponseWriter in case
// an error response is needed as result to the HTTP request
func errorJson(w http.ResponseWriter, c int, m string, d interface{}) {
	l := log.WithFields(log.Fields{
		"action": "http_error.ErrorJson",
	})
	l.Debugf("Request failed with code %d (%s): %s", c, HttpStatusMsg[c], m)
	errorJson := ErrorResponseJson{
		StatusCode:   c,
		Status:       HttpStatusMsg[c],
		ErrorMessage: m,
		ErrorData:    d,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	if err := json.NewEncoder(w).Encode(errorJson); err != nil {
		l.Errorf("Failed to write error response JSON: %s", err)
	}
}
