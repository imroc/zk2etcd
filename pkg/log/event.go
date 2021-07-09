package log

import (
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type Event struct {
	logger  *zap.SugaredLogger
	eventId string
}

func newEvent(logger *zap.SugaredLogger) *Event {
	return &Event{
		logger: logger.With("type", "event", "traceId", generateEventID()),
	}
}

func generateEventID() string {
	return uuid.NewV4().String()
}

func (e *Event) Record(msg string, keysAndValues ...interface{}) {
	if e == nil {
		return
	}
	e.logger.Infow(msg, keysAndValues...)
}
