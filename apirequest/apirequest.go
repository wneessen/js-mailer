package apirequest

import (
	"github.com/ReneKroon/ttlcache/v2"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/config"
	"github.com/wneessen/js-mailer/response"
	"net/http"
)

type ApiRequest struct {
	Cache   *ttlcache.Cache
	Config  *config.Config
	IsHttps bool
	Scheme  string
}

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

	switch r.URL.String() {
	case "/api/v1/token":
		a.GetToken(w, r)
		return
	case "/api/v1/send":
		a.SendForm(w, r)
		return
	default:
		response.ErrorJson(w, 404, "Not found")
	}
}
