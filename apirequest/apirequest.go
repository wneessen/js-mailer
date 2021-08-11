package apirequest

import (
	"github.com/ReneKroon/ttlcache/v2"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/config"
	"github.com/wneessen/js-mailer/form"
	"github.com/wneessen/js-mailer/response"
	"net/http"
	"strings"
)

// ApiRequest reflects a new Api request object
type ApiRequest struct {
	Cache   *ttlcache.Cache
	Config  *config.Config
	IsHttps bool
	Scheme  string
	FormId  string
	Token   string
	FormObj *form.Form
}

// RequestHandler handles an incoming HTTP request on the API routes and
// routes them accordingly to its request type
func (a *ApiRequest) RequestHandler(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"action": "apiRequest.RequestHandler",
	})

	remoteAddr := r.RemoteAddr
	if r.Header.Get("X-Forwarded-For") != "" {
		remoteAddr = r.Header.Get("X-Forwarded-For")
	}
	if r.Header.Get("X-Real-Ip") != "" {
		remoteAddr = r.Header.Get("X-Real-Ip")
	}
	l.Infof("New request to %s from %s", r.URL.String(), remoteAddr)

	a.Scheme = "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		a.IsHttps = true
		a.Scheme = "https"
	}

	// Set general response header
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch {
	case r.URL.String() == "/api/v1/token":
		a.GetToken(w, r)
		return
	case strings.HasPrefix(r.URL.String(), "/api/v1/send/"):
		code, err := a.SendFormParse(r)
		if err != nil {
			l.Errorf("Failed to parse send request: %s", err)
			response.ErrorJsonData(w, code, "Failed parsing send request", err.Error())
			return
		}
		code, err = a.SendFormValidate(r)
		if err != nil {
			response.ErrorJsonData(w, code, "Validation failed", err.Error())
			return
		}
		a.SendForm(w, r)
		return
	default:
		response.ErrorJson(w, 404, "Unknown API route")
	}
}
