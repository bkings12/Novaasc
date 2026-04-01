package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a zap logger from level and format.
func New(level, format string) (*zap.Logger, error) {
	var cfg zap.Config
	switch format {
	case "json":
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
	default:
		cfg = zap.NewDevelopmentConfig()
		cfg.Encoding = "console"
	}

	lvl := zapcore.InfoLevel
	_ = lvl.UnmarshalText([]byte(level))
	cfg.Level = zap.NewAtomicLevelAt(lvl)

	return cfg.Build()
}
