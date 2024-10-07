package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/wneessen/js-mailer/response"

	"github.com/jellydator/ttlcache/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/wneessen/js-mailer/config"
)

// VERSION is the global version string contstant
const VERSION = "0.3.6"

// Srv represents the server object
type Srv struct {
	Cache  *ttlcache.Cache
	Config *config.Config
	Echo   *echo.Echo
}

// Start initializes and starts the web service
func (s *Srv) Start() {
	s.Echo = echo.New()

	// Settings
	s.Echo.HideBanner = true
	s.Echo.HidePort = true
	s.Echo.Debug = s.Config.Loglevel == "debug"
	s.Echo.Server.ReadTimeout = s.Config.Server.Timeout
	s.Echo.Server.WriteTimeout = s.Config.Server.Timeout
	s.Echo.IPExtractor = echo.ExtractIPFromRealIPHeader()
	s.Echo.HTTPErrorHandler = response.CustomError
	s.LogLevel()

	// Register routes
	s.RouterAPI()

	// Middlewares
	s.Echo.Use(middleware.Recover())
	s.Echo.Use(middleware.Logger())
	s.Echo.Use(middleware.BodyLimit(s.Config.Server.RequestLimit))
	s.Echo.Use(middleware.CORS())

	// Start server
	go func() {
		s.Echo.Logger.Infof("Starting js-mailer v%s on: %s", VERSION,
			fmt.Sprintf("%s:%d", s.Config.Server.Addr, s.Config.Server.Port))
		err := s.Echo.Start(fmt.Sprintf("%s:%d", s.Config.Server.Addr, s.Config.Server.Port))
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Echo.Logger.Errorf("Failed to start up web service: %s", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quitSig := make(chan os.Signal, 1)
	signal.Notify(quitSig, os.Interrupt)
	<-quitSig
	shutdownCtx, ctxCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer ctxCancel()
	if err := s.Echo.Shutdown(shutdownCtx); err != nil {
		s.Echo.Logger.Errorf("Failed to shut down gracefully: %s", err)
		os.Exit(1)
	}
}

// LogLevel sets the log level based on the given loglevel in the config object
func (s *Srv) LogLevel() {
	switch s.Config.Loglevel {
	case "debug":
		s.Echo.Logger.SetLevel(log.DEBUG)
	case "info":
		s.Echo.Logger.SetLevel(log.INFO)
	case "warn":
		s.Echo.Logger.SetLevel(log.WARN)
	case "error":
		s.Echo.Logger.SetLevel(log.ERROR)
	default:
		s.Echo.Logger.SetLevel(log.INFO)
	}
}
