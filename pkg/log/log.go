package log

import (
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func GetLogger() *zap.Logger {
	return logger.Desugar()
}

func Init(opt *Options) {
	if opt == nil {
		return
	}
	logger = opt.buildLogger().Sugar()
}

