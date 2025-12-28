// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

type Logger struct {
	*slog.Logger
}

func New(level slog.Level, format string) *Logger {
	return NewLogger(level, format, os.Stderr)
}

func NewLogger(level slog.Level, format string, output io.Writer) *Logger {
	switch strings.ToLower(format) {
	case "text":
		return &Logger{slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level}))}
	default:
		return &Logger{slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level}))}
	}
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

func RequestID(r *http.Request) slog.Attr {
	return slog.String("request_id", middleware.GetReqID(r.Context()))
}
