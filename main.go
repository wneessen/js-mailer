package main

import (
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	log "github.com/sirupsen/logrus"
	"github.com/wneessen/js-mailer/apirequest"
	"github.com/wneessen/js-mailer/config"
	"github.com/wneessen/js-mailer/logging"
	"net/http"
	"os"
	"time"
)

func main() {
	// Initialize everything
	logging.SetLogging()
	confObj := config.NewConfig()
	logging.SetLogLevel(confObj.Loglevel)
	serve(&confObj)
}

func serve(c *config.Config) {
	l := log.WithFields(log.Fields{
		"action": "main.serve",
	})
	l.Infof("Starting up js-mailer server API on port %s:%d", c.Api.Addr, c.Api.Port)

	// Initialize the cache
	cacheObj := ttlcache.NewCache()
	if err := cacheObj.SetTTL(time.Duration(10 * time.Minute)); err != nil {
		l.Errorf("Failed to set TTL on cache object: %s", err)
	}
	defer func() {
		if err := cacheObj.Close(); err != nil {
			l.Errorf("Failed to close cache object: %s", err)
		}
	}()

	// Initalize the Api request object
	apiReq := &apirequest.ApiRequest{
		Cache:  cacheObj,
		Config: c,
	}
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", apiReq.RequestHandler)
	httpSrv := &http.Server{
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           http.TimeoutHandler(httpMux, time.Second*15, ""),
		Addr:              fmt.Sprintf("%s:%d", c.Api.Addr, c.Api.Port),
	}
	if err := httpSrv.ListenAndServe(); err != nil {
		l.Errorf("Failed to start server: %s", err)
		os.Exit(1)
	}
}