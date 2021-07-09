package log

import (
	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
	eventLogger *zap.SugaredLogger
}

var logger *Logger

func GetLogger() *zap.Logger {
	return logger.Desugar()
}

func Init(opt *Options) {
	if opt == nil {
		return
	}
	logger = opt.buildLogger()
}

func (l *Logger) NewEvent() *Event {
	return newEvent(l.eventLogger)
}

func NewEvent() *Event {
	return logger.NewEvent()
}
