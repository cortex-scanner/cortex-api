package main

import (
	"context"
	"cortex/logging"
	"cortex/repository"
	"cortex/service"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type AppConfig struct {
	ListenAddress            string     `env:"CORTEX_LISTEN_ADDRESS"`
	LogLevel                 slog.Level `env:"CORTEX_LOG_LEVEL"`
	Environment              string     `env:"CORTEX_ENVIRONMENT"`
	CORSOrigin               string     `env:"CORTEX_CORS_ALLOWED_ORIGIN"`
	PostgresConnectionString string     `env:"CORTEX_POSTGRES_CONNECTION_STRING"`
	// format should be id.secret with id being a 4 byte hex string and secret being a 16 byte hex string
	AgentToken string `env:"CORTEX_AGENT_TOKEN"`
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

	// connect to database
	pool := setupDatabase(appConfig, logger)

	// setup services
	scanRepo := repository.NewPostgresScanRepository()
	authRepo := repository.NewPostgresAuthRepository()
	agentRepo := repository.NewPostgresAgentRepository()

	scanService := service.NewScanService(scanRepo, pool)
	authService := service.NewAuthService(authRepo, agentRepo, pool)
	agentService := service.NewAgentService(agentRepo, pool)

	// create initial agent if specified
	if appConfig.AgentToken != "" {
		_, err := agentService.CreateAgentWithToken(context.Background(), appConfig.AgentToken, "Default")
		if err != nil {
			logger.Error("failed to create default agent", logging.FieldError, err)
			os.Exit(1)
		}
	}

	// start api server
	serverOptions := ServerOptions{
		ListenAddress: appConfig.ListenAddress,
		CorsOrigin:    appConfig.CORSOrigin,
		ScanService:   scanService,
		AuthService:   authService,
		AgentService:  agentService,
	}

	logger.Debug("allowed CORS origin: " + appConfig.CORSOrigin)

	server := NewServer(serverOptions)
	server.Start()
}

func setupDatabase(appConfig AppConfig, logger *slog.Logger) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), appConfig.PostgresConnectionString)
	if err != nil {
		logger.Error("failed to parse database connection string", logging.FieldError, err)
		os.Exit(1)
	}

	// try database connection
	var test string
	err = pool.QueryRow(context.Background(), "SELECT 'test'").Scan(&test)
	if err != nil {
		logger.Error("failed to connect to database", logging.FieldError, err)
		os.Exit(1)
	}
	logger.Debug("connected to database")

	return pool
}
