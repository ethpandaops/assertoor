package logger

import (
	"maps"

	"github.com/sirupsen/logrus"
)

type logEventBusHook struct {
	logger   *LogScope
	eventBus EventBusPublisher
}

func newLogEventBusHook(logger *LogScope, eventBus EventBusPublisher) *logEventBusHook {
	return &logEventBusHook{
		logger:   logger,
		eventBus: eventBus,
	}
}

func (h *logEventBusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *logEventBusHook) Fire(entry *logrus.Entry) error {
	if h.eventBus == nil {
		return nil
	}

	// Convert logrus fields to map[string]any
	var fields map[string]any
	if len(entry.Data) > 0 {
		fields = make(map[string]any, len(entry.Data))
		maps.Copy(fields, entry.Data)
	}

	h.eventBus.PublishTaskLog(
		h.logger.options.TestRunID,
		h.logger.options.TaskID,
		h.logger.options.TaskName,
		h.logger.options.TaskRefID,
		entry.Level.String(),
		entry.Message,
		fields,
	)

	return nil
}
