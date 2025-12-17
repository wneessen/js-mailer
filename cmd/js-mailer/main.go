// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
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

	// We start with a very basic logger
	log := logger.New(slog.LevelError)

	// Read default config
	conf, err := config.New()
	if err != nil {
		log.Error("failed to load config", logger.Err(err))
		os.Exit(1)
	}

	// Check if config file was specified
	confPath := flag.String("config", "", "path to the config file")
	flag.Parse()
	if *confPath != "" {
		file := filepath.Base(*confPath)
		path := filepath.Dir(*confPath)
		conf, err = config.NewFromFile(path, file)
		if err != nil {
			log.Error("failed to load config from file", logger.Err(err))
			os.Exit(1)
		}
	}

	// Initialize a new logger with the config
	log = logger.New(conf.Log.Level)

	// Initalize server instance
	srv := server.New(conf, log)

	// Start server
	log.Info("starting js-mailer service", slog.String("version", version),
		slog.String("commit", commit), slog.String("date", date))
	if err = srv.Start(ctx); err != nil {
		log.Error("failed to start server", logger.Err(err))
	}
	log.Info("shutting down js-mailer service")
}
