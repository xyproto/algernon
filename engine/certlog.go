package engine

import (
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// certMagicLogger returns a logger for CertMagic that passes the log messages
// on to logrus, so that they are formatted like the rest of the log messages
// and end up in the log file, if one is used
func certMagicLogger() *zap.Logger {
	return zap.New(&logrusCore{})
}

// zapLevel converts a logrus log level to the corresponding zap log level
func zapLevel(level logrus.Level) zapcore.Level {
	switch level {
	case logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel:
		return zapcore.ErrorLevel
	case logrus.WarnLevel:
		return zapcore.WarnLevel
	case logrus.InfoLevel:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}

// logrusCore is a zapcore.Core that logs with logrus
type logrusCore struct {
	fields []zapcore.Field
}

func (c *logrusCore) Enabled(level zapcore.Level) bool {
	return level >= zapLevel(logrus.GetLevel())
}

func (c *logrusCore) With(fields []zapcore.Field) zapcore.Core {
	combined := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	combined = append(combined, c.fields...)
	combined = append(combined, fields...)
	return &logrusCore{fields: combined}
}

func (c *logrusCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *logrusCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	encoded := zapcore.NewMapObjectEncoder()
	for _, field := range c.fields {
		field.AddTo(encoded)
	}
	for _, field := range fields {
		field.AddTo(encoded)
	}
	if entry.LoggerName != "" {
		encoded.Fields["logger"] = entry.LoggerName
	}
	e := logrus.WithFields(logrus.Fields(encoded.Fields))
	switch {
	case entry.Level >= zapcore.ErrorLevel: // also for panic and fatal, since CertMagic should not end the process
		e.Error(entry.Message)
	case entry.Level == zapcore.WarnLevel:
		e.Warn(entry.Message)
	case entry.Level == zapcore.InfoLevel:
		e.Info(entry.Message)
	default:
		e.Debug(entry.Message)
	}
	return nil
}

func (c *logrusCore) Sync() error {
	return nil
}
