package server

import "net/http"

func (s *Server) serverHeader(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Del("Server")
		w.Header().Set("Server", "js-mailer/"+Version)
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
