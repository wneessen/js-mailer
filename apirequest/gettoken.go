package apirequest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"time"
)

type TokenResponseJson struct {
	Token      string `json:"token"`
	CreateTime int64  `json:"create_time,omitempty"`
	ExpireTime int64  `json:"expire_time,omitempty"`
	FormId     int    `json:"form_id"`
	Url        string `json:"url"`
}

func (a *ApiRequest) GetToken(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"action": "apiRequest.GetToken",
	})

	var formId string
	if err := r.ParseMultipartForm(a.Config.Forms.MaxLength); err != nil {
		l.Errorf("Failed to parse form parameters: %s", err)
		response.ErrorJson(w, 500, "Internal Server Error")
		return
	}
	formId = r.Form.Get("formid")
	if formId == "" {
		response.ErrorJson(w, 400, "Bad Request")
		return
	}

	// Let's try to read formobj from cache
	formObj, err := a.GetForm(formId)
	if err != nil {
		l.Errorf("Failed get formObj: %s", err)
		response.ErrorJson(w, 500, "Internal Server Error")
		return
	}

	// Let's validate the Origin header
	isValid := false
	reqOrigin := r.Header.Get("origin")
	if reqOrigin == "" {
		l.Errorf("No origin domain set in HTTP request")
		response.ErrorJson(w, 401, "Unauthorized")
		return
	}
	for _, d := range formObj.Domains {
		if reqOrigin == d || reqOrigin == fmt.Sprintf("http://%s", d) ||
			reqOrigin == fmt.Sprintf("https://%s", d) {
			isValid = true
		}
	}
	if !isValid {
		l.Errorf("Domain %q not in allowed domains list for form %d", reqOrigin, formObj.Id)
		response.ErrorJson(w, 401, "Unauthorized")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", reqOrigin)

	// Generate the token
	nowTime := time.Now()
	expTime := time.Now().Add(time.Minute * 10)
	tokenText := fmt.Sprintf("%s_%d_%d_%d_%s", reqOrigin, nowTime.Unix(), expTime.Unix(), formObj.Id, formObj.Secret)
	tokenSha := fmt.Sprintf("%x", sha256.Sum256([]byte(tokenText)))
	respToken := TokenResponseJson{
		Token:      tokenSha,
		CreateTime: nowTime.Unix(),
		ExpireTime: expTime.Unix(),
		FormId:     formObj.Id,
		Url:        fmt.Sprintf("%s://%s/api/v1/send", a.Scheme, r.Host),
	}
	if err := a.Cache.Set(tokenSha, respToken); err != nil {
		l.Errorf("Failed to store response token in cache: %s", err)
		response.ErrorJson(w, 500, "Internal Server Error")
		return
	}
	if err := json.NewEncoder(w).Encode(respToken); err != nil {
		l.Errorf("Failed to encode response token JSON: %s", err)
		response.ErrorJson(w, 500, "Internal Server Error")
		return
	}
}
