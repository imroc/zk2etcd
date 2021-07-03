package log

import (
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.SugaredLogger

func GetLogger() *zap.Logger {
	return logger.Desugar()
}

func Init(opt *Option) {
	if opt == nil {
		return
	}
	logger = opt.buildLogger().Sugar()
}

type Option struct {
	LogLevel string
}

func AddFlags(fs *flag.FlagSet) *Option {
	var opt Option
	opt.AddFlags(fs)
	return &opt
}

func (opt *Option) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&opt.LogLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")
}

func (opt *Option) buildLogger() *zap.Logger {

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
	return log
}
