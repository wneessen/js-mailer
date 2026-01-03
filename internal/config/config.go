// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/kkyr/fig"
)

const configEnv = "JSMAILER"

// Config represents the global config object struct
type Config struct {
	Cache struct {
		Type     string        `fig:"type" default:"inmemory"`
		Lifetime time.Duration `fig:"lifetime" default:"10m"`
	}
	Log struct {
		Level     slog.Level `fig:"level" default:"0"`
		Format    string     `fig:"format" default:"json"`
		DontLogIP bool       `fig:"dont_log_ip"`
	}

	Forms struct {
		Path              string        `fig:"path" validate:"required"`
		DefaultExpiration time.Duration `fig:"default_expiration" default:"10m"`
	} `fig:"forms"`

	Server struct {
		BindAddress string        `fig:"address" default:"127.0.0.1"`
		BindPort    string        `fig:"port" default:"8765"`
		Timeout     time.Duration `fig:"timeout" default:"15s"`
	} `fig:"server"`
}

// New returns a new Config. It tries to load the config from the default location
// and falls back to the defaults or environment variables if the config file
// was not found.
func New() (*Config, error) {
	conf := new(Config)

	configPath, configFile := findConfigFile()
	if configPath != "" && configFile != "" {
		return NewFromFile(configPath, configFile)
	}

	if err := fig.Load(conf, fig.AllowNoFile(), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load Config: %w", err)
	}

	return conf, nil
}

// NewFromFile returns a new Config from the given path and file.
func NewFromFile(path, file string) (*Config, error) {
	conf := new(Config)
	_, err := os.Stat(filepath.Join(path, file))
	if err != nil {
		return conf, fmt.Errorf("failed to read Config: %w", err)
	}

	if err = fig.Load(conf, fig.Dirs(path), fig.File(file), fig.UseEnv(configEnv)); err != nil {
		return conf, fmt.Errorf("failed to load Config: %w", err)
	}

	return conf, nil
}

func findConfigFile() (string, string) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}
	exts := []string{"toml", "yaml", "yml", "json"}
	for _, ext := range exts {
		path := filepath.Join(homedir, ".config", "js-mailer", "js-mailer."+ext)
		if _, err = os.Stat(path); err == nil {
			return filepath.Dir(path), filepath.Base(path)
		}
	}
	return "", ""
}
