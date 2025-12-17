// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wneessen/js-mailer/internal/forms"
	"github.com/wneessen/js-mailer/internal/logger"
)

const AccessControlMaxAge = "600"

func (s *Server) preflightCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		formID := chi.URLParam(r, "formID")
		if formID == "" {
			s.log.Warn("missing form ID", slog.Any("headers", r.Header), slog.String("origin", origin),
				slog.String("path", r.URL.Path), slog.String("method", r.Method))
			next.ServeHTTP(w, r)
			return
		}
		form, err := forms.New(s.config.Forms.Path, formID)
		if err != nil || form == nil {
			s.log.Error("failed to load form configuration", logger.Err(err), slog.String("formID", formID))
			next.ServeHTTP(w, r)
			return
		}

		allowedDomain := false
		for _, domain := range form.Domains {
			if strings.EqualFold(origin, fmt.Sprintf("https://%s", domain)) {
				allowedDomain = true
				break
			}
		}
		if !allowedDomain {
			s.log.Warn("origin not allowed", slog.String("origin", origin), slog.String("form", formID))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// must be set for all CORS responses
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")

		// Set CORS headers for preflight requests
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			w.Header().Set("Access-Control-Max-Age", AccessControlMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
