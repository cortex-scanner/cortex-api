package logging

import (
	"log/slog"
)

type LoggerComponent string

const (
	API        LoggerComponent = "api"
	Audit      LoggerComponent = "audit"
	DataAccess LoggerComponent = "dal"
	Scan       LoggerComponent = "scan"
)

func GetLogger(component LoggerComponent) *slog.Logger {
	return slog.Default().With("component", component)
}
