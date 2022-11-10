package api

import (
	"github.com/jellydator/ttlcache/v2"
	"github.com/wneessen/js-mailer/config"
)

// Route represents an API route object
type Route struct {
	Cache  *ttlcache.Cache
	Config *config.Config
}
