package validation

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
)

// HcaptchaResponseJson reflect the API response from hCaptcha
type HcaptchaResponseJson struct {
	Success            bool   `json:"success"`
	ChallengeTimestamp string `json:"challenge_ts"`
	Hostname           string `json:"hostname"`
}

// Hcaptcha validates the hCaptcha challenge against the hCaptcha API
func Hcaptcha(c, s string) bool {
	l := log.WithFields(log.Fields{
		"action": "validation.Hcaptcha",
	})

	// Create a HTTP request
	postData := url.Values{
		"response": {c},
		"secret":   {s},
	}
	httpResp, err := http.PostForm("https://hcaptcha.com/siteverify", postData)
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
		var hcapResp HcaptchaResponseJson
		if err := json.Unmarshal(respBody.Bytes(), &hcapResp); err != nil {
			l.Errorf("Failed to unmarshal response JSON: %s", err)
			return false
		}
		return hcapResp.Success
	}

	return false
}
