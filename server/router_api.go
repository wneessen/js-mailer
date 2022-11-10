package server

import (
	"github.com/wneessen/js-mailer/api"
)

// RouterAPI registers the JSON API routes with echo
func (s *Srv) RouterAPI() {
	apiRoute := api.Route{
		Cache:  s.Cache,
		Config: s.Config,
	}
	ag := s.Echo.Group("/api/v1")

	// API routes
	ag.Add("GET", "/ping", apiRoute.Ping)
	ag.Add("GET", "/token", apiRoute.GetToken)
	ag.Add("POST", "/token", apiRoute.GetToken)
	ag.Add("POST", "/send/:fid/:token", apiRoute.SendForm,
		apiRoute.SendFormBindForm, apiRoute.SendFormReqFields, apiRoute.SendFormHoneypot,
		apiRoute.SendFormHcaptcha, apiRoute.SendFormRecaptcha, apiRoute.SendFormCheckToken)
}
