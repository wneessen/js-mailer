// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

func (s *Server) routes(_ context.Context) error {
	logFormat := httplog.SchemaECS
	logSkipPath := []string{"/skip"}
	logger := s.log.With(slog.String("service", "http"))
	logHandler := httplog.RequestLogger(
		logger,
		&httplog.Options{
			Level: s.config.Log.Level,
			Skip: func(req *http.Request, code int) bool {
				for _, skip := range logSkipPath {
					if strings.HasPrefix(req.URL.Path, skip) && code == 200 {
						return true
					}
				}
				return false
			},
			Schema:        logFormat,
			RecoverPanics: true,
		},
	)

	// Register middleware
	s.mux.Use(middleware.RealIP)
	s.mux.Use(middleware.StripSlashes)
	s.mux.Use(middleware.Compress(5))
	s.mux.Use(logHandler)

	// Register routes
	s.mux.Get("/ping", s.HandlerAPIPingGet)

	// Preflight check routes
	s.mux.With(s.preflightCheck).Route("/", func(r chi.Router) {
		r.Get("/token/{formID}", s.HandlerAPITokenGet)
		r.Options("/token/{formID}", s.HandlerAPITokenGet)

		r.Post("/send/{formID}/{hash}", s.HandlerAPISendFormPost)
		r.Options("/send/{formID}/{hash}", s.HandlerAPISendFormPost)
	})

	return nil
}
