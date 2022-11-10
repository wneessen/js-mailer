package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kkyr/fig"
)

// Config represents the global config object struct
type Config struct {
	Loglevel string `fig:"loglevel" default:"debug"`
	Forms    struct {
		Path string `fig:"path" validate:"required"`
	}
	Server struct {
		Addr         string        `fig:"bind_addr"`
		Port         uint32        `fig:"port" default:"8765"`
		Timeout      time.Duration `fig:"timeout" default:"15s"`
		RequestLimit string        `fig:"request_limit" default:"10M"`
	}
}

// NewConfig returns a new Config object and fails if the configuration was not found or
// has bad syntax
func NewConfig() *Config {
	lf := log.Lmsgprefix | log.LstdFlags
	el := log.New(os.Stderr, "[ERROR] ", lf)

	confFileName := "js-mailer.json"
	confFilePath := "/etc/js-mailer/"
	if len(os.Args) == 2 {
		confFileName = filepath.Base(os.Args[1])
		confFilePath = filepath.Dir(os.Args[1])
	}
	_, err := os.Stat(fmt.Sprintf("%s/%s", confFilePath, confFileName))
	if err != nil {
		el.Printf("Failed to read config file %s/%s: %s", confFilePath, confFileName, err)
		os.Exit(1)
	}

	var confObj Config
	if err := fig.Load(&confObj, fig.File(confFileName), fig.Dirs(confFilePath)); err != nil {
		el.Printf("Failed to read config from file: %s", err)
		os.Exit(1)
	}

	return &confObj
}
