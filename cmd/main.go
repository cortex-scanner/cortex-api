package main

import (
	"cortex/logging"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/lmittmann/tint"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type AppConfig struct {
	ListenAddress string     `env:"CORTEX_LISTEN_ADDRESS"`
	LogLevel      slog.Level `env:"CORTEX_LOG_LEVEL"`
	Environment   string     `env:"CORTEX_ENVIRONMENT"`
	CORSOrigin    string     `env:"CORTEX_CORS_ALLOWED_ORIGIN"`
}

func main() {
	// load environment variables
	var appConfig = AppConfig{
		ListenAddress: ":3001",
		LogLevel:      slog.LevelDebug,
		Environment:   EnvProd,
		CORSOrigin:    "*",
	}
	if err := env.Parse(&appConfig); err != nil {
		fmt.Println(err)
		panic("Error loading environment variables")
	}

	// setup logging
	w := os.Stdout
	var logger *slog.Logger
	if appConfig.Environment == EnvDev {
		// pretty log to console
		//nolint:exhaustruct // pkg defaults are fine
		loggerOptions := &tint.Options{
			Level: appConfig.LogLevel,
		}
		logger = slog.New(&logging.ContextHandler{Handler: tint.NewHandler(w, loggerOptions)})
	} else {
		// log json
		//nolint:exhaustruct // pkg defaults are fine
		loggerOptions := &slog.HandlerOptions{
			Level: appConfig.LogLevel,
		}
		logger = slog.New(&logging.ContextHandler{Handler: slog.NewJSONHandler(w, loggerOptions)})
	}

	slog.SetDefault(logger)

	// start api server
	serverOptions := ServerOptions{
		ListenAddress: appConfig.ListenAddress,
		CorsOrigin:    appConfig.CORSOrigin,
	}

	logger.Debug("allowed CORS origin: " + appConfig.CORSOrigin)

	server := NewServer(serverOptions)
	server.Start()
}
