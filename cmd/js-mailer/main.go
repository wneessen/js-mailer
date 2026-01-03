// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/wneessen/js-mailer/internal/config"
	"github.com/wneessen/js-mailer/internal/logger"
	"github.com/wneessen/js-mailer/internal/server"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGKILL,
		syscall.SIGABRT, os.Interrupt)
	defer cancel()

	var conf *config.Config
	var err error

	confPath := flag.String("config", "", "path to the config file")
	flag.Parse()
	switch {
	case confPath != nil && *confPath != "":
		file := filepath.Base(*confPath)
		path := filepath.Dir(*confPath)
		conf, err = config.NewFromFile(path, file)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to load config from file: %s\n", err)
			os.Exit(1)
		}
	default:
		conf, err = config.New()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to load default config: %s\n", err)
			os.Exit(1)
		}
	}

	// Initialize a logger based on the config
	log := logger.New(conf.Log.Level, logger.Opts{Format: conf.Log.Format, DontLogIP: conf.Log.DontLogIP})

	// Initialize server instance
	srv := server.New(conf, log, version)

	// Start server
	log.Info("starting js-mailer service", slog.String("version", version),
		slog.String("commit", commit), slog.String("date", date))
	if err = srv.Start(ctx); err != nil {
		log.Error("failed to start server", logger.Err(err))
	}
	log.Info("shutting down js-mailer service")
}
