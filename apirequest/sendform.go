package apirequest

import (
	"crypto/sha256"
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/go-mail/mail"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"strings"
	"time"
)

// SentSuccessfullJson represents a send confirmation JSON struct
type SentSuccessfullJson struct {
	FormId   string `json:"form_id"`
	SendTime int64  `json:"send_time"`
}

// SendFormParse parses the coming form data of a send requests and returns an error
// if data is missing or incorrect
func (a *ApiRequest) SendFormParse(r *http.Request) (int, error) {
	urlParts := strings.SplitN(r.URL.String(), "/", 6)
	if len(urlParts) != 6 {
		return 404, fmt.Errorf("Invalid URL")
	}
	a.FormId = urlParts[4]
	a.Token = urlParts[5]

	// Only if the URL is syntatically correct, let's parse the body
	if err := r.ParseMultipartForm(a.Config.Forms.MaxLength); err != nil {
		return 500, err
	}
	return 0, nil
}

// SendFormValidate validates that all requirement are fullfilled and returns an error
// if the validation failed
func (a *ApiRequest) SendFormValidate(r *http.Request) (int, error) {
	l := log.WithFields(log.Fields{
		"action": "apiRequest.SendFormValidate",
	})

	// Let's retrieve the formObj from cache
	var tokenRespObj TokenResponseJson
	cacheObj, err := a.Cache.Get(a.Token)
	if err == ttlcache.ErrNotFound {
		return 404, fmt.Errorf("Not a valid send URL")
	}
	if err != nil && err != ttlcache.ErrNotFound {
		return 500, fmt.Errorf("Failed to look up token in cache: %s", err)
	}
	if cacheObj != nil {
		tokenRespObj = cacheObj.(TokenResponseJson)
	}
	if tokenRespObj.FormId != a.FormId {
		l.Warn("URL form id does not match the cached form object id")
		return 400, fmt.Errorf("Invalid form id")
	}
	defer func() {
		if err := a.Cache.Remove(a.Token); err != nil {
			l.Errorf("Failed to delete used token from cache: %s", err)
		}
	}()

	// Let's try to read formobj from cache
	formObj, err := a.GetForm(fmt.Sprintf("%d", a.FormId))
	if err != nil {
		l.Errorf("Failed get formObj: %s", err)
		return 500, fmt.Errorf("Form lookup failed")
	}
	a.FormObj = &formObj

	// Make sure all required fields are set
	// Maybe we can build some kind of validator here
	missingFields := []string{}
	for _, f := range formObj.Content.RequiredFields {
		if r.Form.Get(f) == "" {
			missingFields = append(missingFields, f)
		}
	}
	if len(missingFields) > 0 {
		l.Errorf("Required fields missing: %s", strings.Join(missingFields, ", "))
		return 400, fmt.Errorf("Required fields missing: %s", strings.Join(missingFields, ", "))
	}

	// Check the token
	reqOrigin := r.Header.Get("origin")
	if reqOrigin == "" {
		l.Errorf("No origin domain set in HTTP request")
		return 401, fmt.Errorf("Domain not allowed to access form")
	}
	tokenText := fmt.Sprintf("%s_%d_%d_%d_%s", reqOrigin, tokenRespObj.CreateTime, tokenRespObj.ExpireTime,
		formObj.Id, formObj.Secret)
	tokenSha := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenText)))
	if tokenSha != a.Token {
		l.Errorf("No origin domain set in HTTP request")
		return 401, fmt.Errorf("Domain not allowed to access form")
	}

	return 0, nil
}

// SendForm handles a send Api request
func (a *ApiRequest) SendForm(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"action": "apiRequest.SendForm",
	})

	// Compose the mail message
	mailMsg := mail.NewMessage()
	mailMsg.SetHeader("From", a.FormObj.Sender)
	mailMsg.SetHeader("To", a.FormObj.Recipients...)
	mailMsg.SetHeader("Subject", a.FormObj.Content.Subject)

	mailBody := "The following form fields have been transmitted:\n"
	for _, k := range a.FormObj.Content.Fields {
		if v := r.Form.Get(k); v != "" {
			mailBody = fmt.Sprintf("%s\n* %s => %s", mailBody, k, v)
		}
	}
	mailMsg.SetBody("text/plain", mailBody)

	// Send the mail message
	var serverTimeout time.Duration
	var err error
	serverTimeout, err = time.ParseDuration(a.FormObj.Server.Timeout)
	if err != nil {
		l.Warnf("Could not parse configured server timeout: %s", err)
		serverTimeout = time.Second * 5
	}
	mailDailer := mail.NewDialer(a.FormObj.Server.Host, a.FormObj.Server.Port, a.FormObj.Server.Username,
		a.FormObj.Server.Password)
	mailDailer.Timeout = serverTimeout
	if a.FormObj.Server.ForceTLS {
		mailDailer.StartTLSPolicy = mail.MandatoryStartTLS
	}
	mailSender, err := mailDailer.Dial()
	if err != nil {
		l.Errorf("Could not connect to configured mail server: %s", err)
		response.ErrorJson(w, 500, err.Error())
		return
	}
	defer func() {
		if err := mailSender.Close(); err != nil {
			l.Errorf("Failed to close mail server connection: %s", err)
		}
	}()
	if err := mail.Send(mailSender, mailMsg); err != nil {
		l.Errorf("Could not send mail message: %s", err)
		response.ErrorJson(w, 500, err.Error())
		return
	}

	response.SuccessJson(w, 200, &SentSuccessfullJson{
		FormId:   a.FormObj.Id,
		SendTime: time.Now().Unix(),
	})
}
