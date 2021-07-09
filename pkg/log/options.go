package log

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel       string
	EnableEventLog bool
}

func (opt *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&opt.LogLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")
	fs.BoolVar(&opt.EnableEventLog, "enable-event-log", false, "enable event log")
}

func (opt *Options) buildLogger() *Logger {
	var lv zapcore.Level
	switch opt.LogLevel {
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
	log, err := config.Build()
	if err != nil {
		panic(err)
	}
	return &Logger{
		log.Sugar(),
		createEventLogger(opt.EnableEventLog),
	}
}

func createEventLogger(enable bool) *zap.SugaredLogger {
	lv := zapcore.InfoLevel
	if !enable {
		lv = zapcore.WarnLevel
	}
	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	enc.LevelKey = zapcore.OmitKey
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
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger.Sugar()
}
