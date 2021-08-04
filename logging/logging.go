package logging

import (
	log "github.com/sirupsen/logrus"
	"strings"
)

func SetLogging() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
		DisableColors:          false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05 -0700",
	})
}

func SetLogLevel(l string) {
	if l == "" {
		return
	}
	switch strings.ToLower(l) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
