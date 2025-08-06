package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func InitLogger() {
	jsonHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	Logger = slog.New(jsonHandler)
}
