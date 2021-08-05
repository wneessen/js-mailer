package apirequest

import (
	"crypto/sha256"
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/go-mail/mail"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/http_error"
	"net/http"
	"strings"
	"time"
)

func (a *ApiRequest) SendForm(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"action": "apiRequest.SendForm",
	})

	var formId string
	if err := r.ParseMultipartForm(a.Config.Forms.MaxLength); err != nil {
		l.Errorf("Failed to parse form parameters: %s", err)
		http_error.ErrorJson(w, 500, "Internal Server Error")
		return
	}
	formId = r.Form.Get("formid")
	if formId == "" {
		http_error.ErrorJson(w, 400, "Bad Request")
		return
	}
	sendToken := r.Form.Get("token")
	if sendToken == "" {
		http_error.ErrorJson(w, 400, "Bad Request")
		return
	}

	// Let's retrieve the formObj from cache
	var tokenRespObj TokenResponseJson
	cacheObj, err := a.Cache.Get(sendToken)
	if err == ttlcache.ErrNotFound {
		http_error.ErrorJson(w, 404, "Not Found")
		return
	}
	if err != nil && err != ttlcache.ErrNotFound {
		http_error.ErrorJson(w, 500, "Internal Server Error")
		return
	}
	if cacheObj != nil {
		tokenRespObj = cacheObj.(TokenResponseJson)
	}
	if fmt.Sprintf("%d", tokenRespObj.FormId) != formId {
		http_error.ErrorJson(w, 400, "Bad Request")
		return
	}
	defer func() {
		if err := a.Cache.Remove(sendToken); err != nil {
			l.Errorf("Failed to delete used token from cache: %s", err)
		}
	}()

	// Let's try to read formobj from cache
	formObj, err := a.GetForm(fmt.Sprintf("%d", tokenRespObj.FormId))
	if err != nil {
		l.Errorf("Failed get formObj: %s", err)
		http_error.ErrorJson(w, 500, "Internal Server Error")
		return
	}

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
		http_error.MissingFieldsJson(w, 400, "Required fields missing", missingFields)
		return
	}

	// Check the token
	reqOrigin := r.Header.Get("origin")
	if reqOrigin == "" {
		l.Errorf("No origin domain set in HTTP request")
		http_error.ErrorJson(w, 401, "Unauthorized")
		return
	}
	tokenText := fmt.Sprintf("%s_%d_%d_%d_%s", reqOrigin, tokenRespObj.CreateTime, tokenRespObj.ExpireTime,
		formObj.Id, formObj.Secret)
	tokenSha := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenText)))
	if tokenSha != sendToken {
		l.Errorf("No origin domain set in HTTP request")
		http_error.ErrorJson(w, 401, "Unauthorized")
		return
	}

	// Compose the mail message
	mailMsg := mail.NewMessage()
	mailMsg.SetHeader("From", formObj.Sender)
	mailMsg.SetHeader("To", formObj.Recipients...)
	mailMsg.SetHeader("Subject", formObj.Content.Subject)

	mailBody := "The following form fields have been transmitted:\n"
	for _, k := range formObj.Content.Fields {
		if v := r.Form.Get(k); v != "" {
			mailBody = fmt.Sprintf("%s\n* %s => %s", mailBody, k, v)
		}
	}
	mailMsg.SetBody("text/plain", mailBody)

	// Send the mail message
	mailDailer := mail.NewDialer(formObj.Server.Host, formObj.Server.Port, formObj.Server.Username,
		formObj.Server.Password)
	mailDailer.Timeout = time.Second * 5
	mailDailer.StartTLSPolicy = mail.OpportunisticStartTLS
	mailSender, err := mailDailer.Dial()
	if err != nil {
		l.Errorf("Could not connect to configured mail server: %s", err)
		http_error.ErrorJson(w, 500, err.Error())
		return
	}
	defer func() { _ = mailSender.Close() }()

	if err := mail.Send(mailSender, mailMsg); err != nil {
		l.Errorf("Could not send mail message: %s", err)
		http_error.ErrorJson(w, 500, err.Error())
		return
	}
}
