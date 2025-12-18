// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"testing"
	"time"
)

const (
	testLogLevel               = 0
	testLogFormat              = "json"
	testFormsPath              = "testdata"
	testFormsDefaultExpiration = time.Minute * 15
	testServerAddress          = "0.0.0.0"
	testServerPort             = "8080"
	testServerCacheLifetime    = time.Minute * 30
	testServerTimeout          = time.Second * 20
)

func TestNew(t *testing.T) {
	t.Run("the default config is returned", func(t *testing.T) {
		t.Setenv("HOME", "../../testdata")
		config, err := New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
		if config.Log.Level != testLogLevel {
			t.Errorf("expected log level to be %d, got %d", testLogLevel, config.Log.Level)
		}
		if config.Log.Format != "json" {
			t.Errorf("expected log format to be %s, got %s", testLogFormat, config.Log.Format)
		}
		if config.Forms.Path != "testdata" {
			t.Errorf("expected forms path to be %s, got %s", testFormsPath, config.Forms.Path)
		}
		if config.Forms.DefaultExpiration != testFormsDefaultExpiration {
			t.Errorf("expected forms default expiration to be %s, got %s", testFormsDefaultExpiration,
				config.Forms.DefaultExpiration)
		}
		if config.Server.BindAddress != testServerAddress {
			t.Errorf("expected server bind address to be %s, got %s", testServerAddress,
				config.Server.BindAddress)
		}
		if config.Server.BindPort != testServerPort {
			t.Errorf("expected server bind port to be %s, got %s", testServerPort, config.Server.BindPort)
		}
		if config.Server.CacheLifetime != testServerCacheLifetime {
			t.Errorf("expected server cache lifetime to be %s, got %s", testServerCacheLifetime,
				config.Server.CacheLifetime)
		}
		if config.Server.Timeout != testServerTimeout {
			t.Errorf("expected server timeout to be %s, got %s", testServerTimeout, config.Server.Timeout)
		}
	})
	t.Run("config without a home directory", func(t *testing.T) {
		t.Setenv("HOME", "")
		_, err := New()
		if err == nil {
			t.Fatal("expected error when no home directory is set")
		}
	})
	t.Run("config with a home directory but no configs", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Cleanup(func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Errorf("failed to remove temp dir: %s", err)
			}
		})

		t.Setenv("HOME", tempDir)
		_, err := New()
		if err == nil {
			t.Fatal("expected error when no home directory is set")
		}
	})
	t.Run("config with a home directory but no configs but enviornment", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Cleanup(func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Errorf("failed to remove temp dir: %s", err)
			}
		})

		t.Setenv("HOME", tempDir)
		t.Setenv("JSMAILER_FORMS_PATH", "testdata")
		_, err := New()
		if err != nil {
			t.Fatalf("failed to create config: %s", err)
		}
	})
}

func TestNewFromFile(t *testing.T) {
	t.Run("return config from file", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			file     string
			succeeds bool
		}{
			{"json", "../../testdata", "config.json", true},
			{"yaml", "../../testdata", "config.yml", true},
			{"toml", "../../testdata/.config/js-mailer", "js-mailer.toml", true},
			{"non-existing", "../../testdata", "non-existing.json", false},
			{"incomplete", "../../testdata", "incomplete.toml", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := NewFromFile(tt.path, tt.file)
				if tt.succeeds && err != nil {
					t.Fatalf("failed to create config from file: %s", err)
				}
			})
		}
	})
}
