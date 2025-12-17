// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wneessen/js-mailer/internal/cache"
	"github.com/wneessen/js-mailer/internal/config"
	"github.com/wneessen/js-mailer/internal/httpclient"
	"github.com/wneessen/js-mailer/internal/logger"
)

type Server struct {
	cache      *cache.Cache
	config     *config.Config
	httpClient *httpclient.Client
	httpSrv    *http.Server
	log        *logger.Logger
	mux        *chi.Mux
}

// New returns a new server instance
func New(conf *config.Config, log *logger.Logger) *Server {
	mux := chi.NewMux()
	listenAddr := net.JoinHostPort(conf.Server.BindAddress, conf.Server.BindPort)

	return &Server{
		cache:      cache.New(conf.Server.CacheLifetime),
		config:     conf,
		httpClient: httpclient.New(log),
		httpSrv: &http.Server{
			Addr:              listenAddr,
			Handler:           mux,
			ReadTimeout:       conf.Server.Timeout,
			ReadHeaderTimeout: conf.Server.Timeout,
			WriteTimeout:      conf.Server.Timeout,
			IdleTimeout:       conf.Server.Timeout,
		},
		log: log,
		mux: mux,
	}
}

// Start starts up the server and waits for a shutdown signal
func (s *Server) Start(ctx context.Context) error {
	s.log.Info("starting js-mailer http server", slog.String("listen_addr", s.httpSrv.Addr))

	// Assign routes
	if err := s.routes(ctx); err != nil {
		return fmt.Errorf("failed to assign routes to muxer: %w", err)
	}

	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("failed to start http listener", logger.Err(err))
		}
	}()
	<-ctx.Done()

	s.log.Info("shutting down js-mailer http server")
	ctxShutdown, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	if err := s.httpSrv.Shutdown(ctxShutdown); err != nil {
		s.log.Error("failed to shut down http server gracefully", logger.Err(err))
	}
	s.cache.Stop()

	return nil
}
