package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/labstack/echo/v4"
	"github.com/wneessen/js-mailer/form"
	"github.com/wneessen/js-mailer/response"
)

// SendFormRequest reflects the structure of the send form request data
type SendFormRequest struct {
	FormId    string `param:"fid"`
	FormObj   *form.Form
	Token     string `param:"token"`
	TokenResp *TokenResponse
}

// CaptchaResponse reflect the API response from various 3rd party captcha services
type CaptchaResponse struct {
	Success            bool   `json:"success"`
	ChallengeTimestamp string `json:"challenge_ts"`
	Hostname           string `json:"hostname"`
}

// HcaptchaResponse is the CaptchaResponse for hCaptcha
type HcaptchaResponse CaptchaResponse

// ReCaptchaResponse is the CaptchaResponse for Google ReCaptcha
type ReCaptchaResponse CaptchaResponse

// SendFormBindForm is a middleware that validates the provided form data and binds
// it to a SendFormRequest object
func (r *Route) SendFormBindForm(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := &SendFormRequest{}
		if err := c.Bind(sr); err != nil {
			c.Logger().Errorf("failed to bind request to SendFormRequest object: %s", err)
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		// Let's retrieve the formObj from cache
		cacheObj, err := r.Cache.Get(sr.Token)
		if err == ttlcache.ErrNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "not a valid send URL")
		}
		if err != nil && err != ttlcache.ErrNotFound {
			c.Logger().Errorf("failed to look up token in cache: %s", err)
			return echo.NewHTTPError(http.StatusInternalServerError, &response.ErrorObj{
				Message: "failed to look up token in cache",
				Data:    err.Error(),
			})
		}
		if cacheObj != nil {
			TokenRespObj := cacheObj.(TokenResponse)
			sr.TokenResp = &TokenRespObj
		}
		if sr.TokenResp != nil && sr.TokenResp.FormId != sr.FormId {
			c.Logger().Warn("URL form id does not match the cached form object id")
			return echo.NewHTTPError(http.StatusBadRequest, "invalid form id")
		}
		defer func() {
			if err := r.Cache.Remove(sr.Token); err != nil {
				c.Logger().Errorf("failed to delete used token from cache: %s", err)
			}
		}()

		// Let's try to read formobj from cache
		formObj, err := r.GetForm(sr.FormId)
		if err != nil {
			c.Logger().Errorf("failed get form object: %s", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "form lookup failed")
		}

		sr.FormObj = &formObj
		c.Set("formobj", sr)

		return next(c)
	}
}

// SendFormReqFields is a middleware that validates that all required fields are set in
// the SendFormRequest object
func (r *Route) SendFormReqFields(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := c.Get("formobj").(*SendFormRequest)
		if sr == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no valid form object found")
		}

		var invalidFields []string
		fieldError := make(map[string]string)
		for _, f := range sr.FormObj.Validation.Fields {
			v := c.FormValue(f.Name)
			if f.Required && v == "" {
				invalidFields = append(invalidFields, f.Name)
				fieldError[f.Name] = "field is required, but missing"
				continue
			}

			switch f.Type {
			case "text":
				continue
			case "email":
				mailRegExp, err := regexp.Compile("^[a-zA-Z0-9.!#$%&'*+/\\=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
				if err != nil {
					c.Logger().Errorf("Failed to compile email comparison regexp: %s", err)
					continue
				}
				if !mailRegExp.Match([]byte(v)) {
					c.Logger().Debugf("Form field is expected to be of type email but does not match this requirementd: %s", f.Name)
					invalidFields = append(invalidFields, f.Name)
					fieldError[f.Name] = "field is expected to be of type email, but does not match"
					continue
				}
			case "number":
				numRegExp, err := regexp.Compile("^[0-9]+$")
				if err != nil {
					c.Logger().Errorf("Failed to compile email comparison regexp: %s", err)
					continue
				}
				if !numRegExp.Match([]byte(v)) {
					c.Logger().Debugf("Form field is expected to be of type number but does not match this requirementd: %s", f.Name)
					invalidFields = append(invalidFields, f.Name)
					fieldError[f.Name] = "field is expected to be of type number, but does not match"
					continue
				}
			case "bool":
				boolRegExp, err := regexp.Compile("^(?i)(true|false|0|1|on|off)$")
				if err != nil {
					c.Logger().Errorf("Failed to compile boolean comparison regexp: %s", err)
					continue
				}
				if !boolRegExp.Match([]byte(v)) {
					c.Logger().Debugf("Form field is expected to be of type boolean but does not match this requirementd: %s", f.Name)
					invalidFields = append(invalidFields, f.Name)
					fieldError[f.Name] = "field is expected to be of type bool, but does not match"
					continue
				}
			case "matchval":
				if v != f.Value {
					invalidFields = append(invalidFields, f.Name)
					fieldError[f.Name] = "field is expected match the configured match value, but isn't"
				}
				continue
			default:
				continue
			}
		}
		if len(invalidFields) > 0 {
			c.Logger().Errorf("Form field validation failed: %s", strings.Join(invalidFields, ", "))
			var errorMsg []string
			for _, f := range invalidFields {
				errorMsg = append(errorMsg, fmt.Sprintf("%s: %s", f, fieldError[f]))
			}
			return echo.NewHTTPError(http.StatusBadRequest, &response.ErrorObj{
				Message: "fields(s) validation failed",
				Data:    fieldError,
			})
		}

		return next(c)
	}
}

// SendFormHoneypot is a middleware that checks that a configured honeypot field is not
// filled with any data
func (r *Route) SendFormHoneypot(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := c.Get("formobj").(*SendFormRequest)
		if sr == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no valid form object found")
		}

		if sr.FormObj.Validation.Honeypot != nil {
			if c.FormValue(*sr.FormObj.Validation.Honeypot) != "" {
				c.Logger().Warnf("form includes a honeypot field which is not empty. Denying request")
				return echo.NewHTTPError(http.StatusBadRequest, "invalid form request data")
			}
		}

		return next(c)
	}
}

// SendFormHcaptcha is a middleware that checks the form data against hCaptcha
func (r *Route) SendFormHcaptcha(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := c.Get("formobj").(*SendFormRequest)
		if sr == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no valid form object found")
		}

		if sr.FormObj.Validation.Hcaptcha.Enabled {
			hcapResponse := c.FormValue("h-captcha-response")
			if hcapResponse == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing hCaptcha response")
			}

			// Create a HTTP request
			postData := url.Values{
				"response": {hcapResponse},
				"secret":   {sr.FormObj.Validation.Hcaptcha.SecretKey},
			}
			httpResp, err := http.PostForm("https://hcaptcha.com/siteverify", postData)
			if err != nil {
				c.Logger().Errorf("failed to post HTTP request to hCaptcha: %s", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "hCaptcha validation failed")
			}

			var respBody bytes.Buffer
			_, err = respBody.ReadFrom(httpResp.Body)
			if err != nil {
				c.Logger().Errorf("reading HTTP response body failed: %s", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "hCaptcha validation failed")
			}
			if httpResp.StatusCode == http.StatusOK {
				var hcapResp HcaptchaResponse
				if err := json.Unmarshal(respBody.Bytes(), &hcapResp); err != nil {
					c.Logger().Errorf("HTTP repsonse JSON unmarshalling failed: %s", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "hCaptcha validation failed")
				}
				if !hcapResp.Success {
					return echo.NewHTTPError(http.StatusBadRequest,
						"hCaptcha challenge-response validation failed")
				}
				return next(c)
			}

			return echo.NewHTTPError(http.StatusBadRequest,
				"hCaptcha challenge-response validation failed")
		}

		return next(c)
	}
}

// SendFormRecaptcha is a middleware that checks the form data against Google ReCaptcha
func (r *Route) SendFormRecaptcha(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := c.Get("formobj").(*SendFormRequest)
		if sr == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no valid form object found")
		}

		if sr.FormObj.Validation.Recaptcha.Enabled {
			recapResponse := c.FormValue("g-recaptcha-response")
			if recapResponse == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing reCaptcha response")
			}

			// Create a HTTP request
			postData := url.Values{
				"response": {recapResponse},
				"secret":   {sr.FormObj.Validation.Recaptcha.SecretKey},
			}
			httpResp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", postData)
			if err != nil {
				c.Logger().Errorf("failed to post HTTP request to reCaptcha: %s", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "reCaptcha validation failed")
			}

			var respBody bytes.Buffer
			_, err = respBody.ReadFrom(httpResp.Body)
			if err != nil {
				c.Logger().Errorf("reading HTTP response body failed: %s", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "reCaptcha validation failed")
			}
			if httpResp.StatusCode == http.StatusOK {
				var recapResp ReCaptchaResponse
				if err := json.Unmarshal(respBody.Bytes(), &recapResp); err != nil {
					c.Logger().Errorf("HTTP repsonse JSON unmarshalling failed: %s", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "reCaptcha validation failed")
				}
				if !recapResp.Success {
					return echo.NewHTTPError(http.StatusBadRequest,
						"reCaptcha challenge-response validation failed")
				}
				return next(c)
			}

			return echo.NewHTTPError(http.StatusBadRequest,
				"reCaptcha challenge-response validation failed")
		}

		return next(c)
	}
}

// SendFormCheckToken is a middleware that checks the form security token
func (r *Route) SendFormCheckToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sr := c.Get("formobj").(*SendFormRequest)
		if sr == nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "no valid form object found")
		}

		reqOrigin := c.Request().Header.Get("origin")
		if reqOrigin == "" {
			c.Logger().Errorf("no origin domain set in HTTP request")
			return echo.NewHTTPError(http.StatusUnauthorized, "domain not allowed to access form")
		}
		tokenText := fmt.Sprintf("%s_%d_%d_%s_%s", reqOrigin, sr.TokenResp.CreateTime,
			sr.TokenResp.ExpireTime, sr.FormObj.Id, sr.FormObj.Secret)
		tokenSha := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenText)))
		if tokenSha != sr.Token {
			c.Logger().Errorf("security token does not match")
			return echo.NewHTTPError(http.StatusUnauthorized, "domain not allowed to access form")
		}

		return next(c)
	}
}
