package main

import (
	"fmt"
	"os"
	"time"

	"github.com/jellydator/ttlcache/v2"
	"github.com/wneessen/js-mailer/config"
	"github.com/wneessen/js-mailer/server"
)

func main() {
	confObj := config.NewConfig()

	cacheObj := ttlcache.NewCache()
	if err := cacheObj.SetTTL(10 * time.Minute); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR: Failed to set default TTL on cache: %s", err)
		os.Exit(1)
	}
	defer func() {
		_ = cacheObj.Close()
	}()

	srv := server.Srv{
		Cache:  cacheObj,
		Config: confObj,
	}
	srv.Start()
}
