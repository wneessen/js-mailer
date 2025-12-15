// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wneessen/js-mailer/internal/forms"
)

const AccessControlMaxAge = "600"

func (s *Server) preflightCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodOptions ||
			r.Header.Get("Origin") == "" ||
			r.Header.Get("Access-Control-Request-Method") == "" {

			next.ServeHTTP(w, r)
			return
		}

		formID := chi.URLParam(r, "formID")
		form, err := forms.New(s.config.Forms.Path, formID)
		if err != nil || form == nil {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		allowedDomain := false
		for _, domain := range form.Domains {
			if strings.EqualFold(origin, fmt.Sprintf("https://%s", domain)) {
				allowedDomain = true
				break
			}
		}
		if !allowedDomain {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
		w.Header().Set("Access-Control-Max-Age", AccessControlMaxAge)
		w.WriteHeader(http.StatusNoContent)
	})
}
