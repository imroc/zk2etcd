package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

func New(level string) *Logger {

	var lv zapcore.Level

	switch level {
	case "debug":
		lv = zapcore.DebugLevel
	case "info":
		lv = zapcore.InfoLevel
	case "warn":
		lv = zapcore.WarnLevel
	case "error":
		lv = zapcore.ErrorLevel
	case "dpanic":
		lv = zapcore.DPanicLevel
	case "panic":
		lv = zapcore.PanicLevel
	case "fatal":
		lv = zapcore.FatalLevel
	default:
		lv = zapcore.InfoLevel
	}

	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	config := &zap.Config{
		Level:       zap.NewAtomicLevelAt(lv),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    enc,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, _ := config.Build()
	sugar := logger.Sugar()
	return &Logger{
		sugar,
	}
}
