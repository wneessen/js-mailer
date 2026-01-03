// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package logger

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

const (
	IPv4HideMask = 16
	IPv6HideMask = 48
)

type Logger struct {
	*slog.Logger
}

type Opts struct {
	Format    string
	DontLogIP bool
}

func New(level slog.Level, opts Opts) *Logger {
	return NewLogger(level, os.Stderr, opts)
}

func NewLogger(level slog.Level, output io.Writer, opts Opts) *Logger {
	replaceattr := func(groups []string, a slog.Attr) slog.Attr {
		return a
	}
	if opts.DontLogIP {
		replaceattr = func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case "client.ip":
				ip := net.ParseIP(a.Value.String())
				switch {
				case ip.To4() != nil:
					ip = ip.Mask(net.CIDRMask(IPv4HideMask, 32))
				case ip.To16() != nil:
					ip = ip.Mask(net.CIDRMask(IPv6HideMask, 128))
				default:
					ip = net.IPv4zero
				}
				return slog.String("client.ip", ip.String())
			}
			return a
		}
	}

	switch strings.ToLower(opts.Format) {
	case "text":
		return &Logger{slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
			ReplaceAttr: replaceattr,
			Level:       level,
		}))}
	default:
		return &Logger{slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{
			ReplaceAttr: replaceattr,
			Level:       level,
		}))}
	}
}

func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

func RequestID(r *http.Request) slog.Attr {
	return slog.String("request_id", middleware.GetReqID(r.Context()))
}
