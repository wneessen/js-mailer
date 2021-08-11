package validation

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

// RecaptchaResponseJson reflect the API response from hCaptcha
type RecaptchaResponseJson struct {
	Success            bool   `json:"success"`
	ChallengeTimestamp string `json:"challenge_ts"`
	Hostname           string `json:"hostname"`
}

// Recaptcha validates the reCaptcha challenge against the Google API
func Recaptcha(c, s string) bool {
	l := log.WithFields(log.Fields{
		"action": "validation.Recaptcha",
	})

	// Create a HTTP request
	postData := url.Values{
		"response": {c},
		"secret":   {s},
	}
	httpResp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", postData)
	if err != nil {
		l.Errorf("an error occurred creating new HTTP POST request: %v", err)
		return false
	}

	var respBody bytes.Buffer
	_, err = respBody.ReadFrom(httpResp.Body)
	if err != nil {
		l.Errorf("Failed to read response body: %s", err)
		return false
	}
	if httpResp.StatusCode == http.StatusOK {
		var recapResp RecaptchaResponseJson
		if err := json.Unmarshal(respBody.Bytes(), &recapResp); err != nil {
			l.Errorf("Failed to unmarshal response JSON: %s", err)
			return false
		}
		return recapResp.Success
	}

	return false
}
