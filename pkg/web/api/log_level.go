package api

import "github.com/sirupsen/logrus"

// mapLogLevelToUI converts logrus levels to UI log level strings.
func mapLogLevelToUI(level uint32) string {
	switch logrus.Level(level) {
	case logrus.TraceLevel:
		return "trace"
	case logrus.DebugLevel:
		return "debug"
	case logrus.InfoLevel:
		return "info"
	case logrus.WarnLevel:
		return "warn"
	case logrus.ErrorLevel:
		return "error"
	case logrus.FatalLevel:
		return "fatal"
	case logrus.PanicLevel:
		return "panic"
	default:
		return "unknown"
	}
}
