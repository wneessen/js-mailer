package config

import (
	"fmt"
	"github.com/kkyr/fig"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

// Config represents the global config object struct
type Config struct {
	Loglevel string `fig:"loglevel" default:"debug"`
	Api      struct {
		Addr string `fig:"bind_addr"`
		Port uint32 `fig:"port" default:"8765"`
	}
	Forms struct {
		Path      string `fig:"path" validate:"required"`
		MaxLength int64  `fig:"maxlength" default:"1024000"`
	}
}

// NewConfig returns a new Config object and fails if the configuration was not found or
// has bad syntax
func NewConfig() Config {
	l := log.WithFields(log.Fields{
		"action": "config.NewConfig",
	})

	confFileName := "js-mailer.json"
	confFilePath := "/etc/js-mailer/"
	if len(os.Args) == 2 {
		confFileName = filepath.Base(os.Args[1])
		confFilePath = filepath.Dir(os.Args[1])
	}
	_, err := os.Stat(fmt.Sprintf("%s/%s", confFilePath, confFileName))
	if err != nil {
		l.Errorf("Failed to read config file %s/%s: %s", confFilePath, confFileName, err)
		os.Exit(1)
	}

	var confObj Config
	if err := fig.Load(&confObj, fig.File(confFileName), fig.Dirs(confFilePath)); err != nil {
		l.Fatalf("Failed to read config from file: %s", err)
		os.Exit(1)
	}

	return confObj
}
