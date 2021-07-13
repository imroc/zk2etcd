package log

import (
	"go.uber.org/zap"
	"net/http"
)

type Logger struct {
	*zap.SugaredLogger
	eventLogger *zap.SugaredLogger
}

var defaultOpt *Options

var logger *Logger

func GetLogger() *zap.Logger {
	return logger.Desugar()
}

func Init(opt *Options) {
	if opt == nil {
		return
	}
	defaultOpt = opt
	logger = defaultOpt.buildLogger()
}

func (l *Logger) NewEvent() *Event {
	return newEvent(l.eventLogger)
}

func NewEvent() *Event {
	return logger.NewEvent()
}

func SetLogLevel(level string) {
	defaultOpt.LogLevel = level
	logger = defaultOpt.buildLogger()
}

func LogLevelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseForm()
	if err != nil {
		Warnw("parse http request error",
			"uri", r.RequestURI,
			"error", err.Error(),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	level := r.FormValue("level")
	SetLogLevel(level)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}
