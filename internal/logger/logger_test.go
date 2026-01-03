// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestNew(t *testing.T) {
	t.Run("return a text logger", func(t *testing.T) {
		for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
			t.Run("log level "+level.String(), func(t *testing.T) {
				log := New(level, Opts{Format: "text"})
				if log == nil {
					t.Fatal("logger is nil")
				}
			})
		}
	})
	t.Run("return a json logger", func(t *testing.T) {
		for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
			t.Run("log level "+level.String(), func(t *testing.T) {
				log := New(level, Opts{Format: "json"})
				if log == nil {
					t.Fatal("logger is nil")
				}
			})
		}
	})
}

func TestNewLogger(t *testing.T) {
	t.Run("log level", func(t *testing.T) {
		tests := []struct {
			name     string
			level    slog.Level
			logDebug bool
			logInfo  bool
			logWarn  bool
			logError bool
		}{
			{"debug", slog.LevelDebug, true, true, true, true},
			{"info", slog.LevelInfo, false, true, true, true},
			{"warn", slog.LevelWarn, false, false, true, true},
			{"error", slog.LevelError, false, false, false, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := bytes.NewBuffer(nil)
				log := NewLogger(tt.level, buf, Opts{Format: "text"})
				if log == nil {
					t.Fatal("logger is nil")
				}
				log.Debug("debug")
				log.Info("info")
				log.Warn("warn")
				log.Error("error")
				if tt.logDebug && !strings.Contains(buf.String(), "level=DEBUG") {
					t.Error("expected logger to log debug messages")
				}
				if tt.logInfo && !strings.Contains(buf.String(), "level=INFO") {
					t.Error("expected logger to log info messages")
				}
				if tt.logWarn && !strings.Contains(buf.String(), "level=WARN") {
					t.Error("expected logger to log warn messages")
				}
				if tt.logError && !strings.Contains(buf.String(), "level=ERROR") {
					t.Error("expected logger to log error messages")
				}
			})
		}
	})
	t.Run("new logger with disabled IP logging", func(t *testing.T) {
		tests := []struct {
			name string
			ip   string
			want string
		}{
			{"IPv4", "192.168.123.123", "192.168.0.0"},
			{"IPv6", "2345:0425:2CA1:0000:0000:0567:5673:23b5", "2345:425:2ca1::"},
			{"No IP given", "invalid", "0.0.0.0"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf := bytes.NewBuffer(nil)
				log := NewLogger(slog.LevelDebug, buf, Opts{Format: "text", DontLogIP: true})
				if log == nil {
					t.Fatal("logger is nil")
				}
				log.Debug("debug", slog.String("client.ip", tt.ip))
				output := strings.TrimSpace(buf.String())
				if !strings.HasSuffix(output, "client.ip="+tt.want) {
					t.Errorf("expected client IP to be %q, got %q", tt.want, output)
				}
			})
		}
	})
	t.Run("new logger with disabled request IP logging and empty string", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		log := NewLogger(slog.LevelDebug, buf, Opts{Format: "text", DontLogIP: true})
		if log == nil {
			t.Fatal("logger is nil")
		}
		log.Debug("debug", "", "")
		output := strings.TrimSpace(buf.String())
		t.Log(output)
	})
}

func TestErr(t *testing.T) {
	t.Run("errors are logged properly", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		log := NewLogger(slog.LevelError, buf, Opts{Format: "text"})
		log.Error("something went wrong", Err(errors.New("test error")))
		want := `level=ERROR msg="something went wrong" error="test error"`
		if !strings.Contains(buf.String(), want) {
			t.Errorf("expected error to contain %q, got %q", want, buf.String())
		}
	})
}

func TestRequestID(t *testing.T) {
	t.Run("request ID is and returned", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(t.Context(), middleware.RequestIDKey, "test")
		log := NewLogger(slog.LevelDebug, buf, Opts{Format: "text"})
		log.Debug("test", RequestID(req.WithContext(ctx)))
		want := `level=DEBUG msg=test request_id=test`
		if !strings.Contains(buf.String(), want) {
			t.Errorf("expected error to contain %q, got %q", want, buf.String())
		}
	})
}
