package x

import "log"

// Logger provides leveled logging for the SDK.
type Logger struct {
	level LogLevel
}

func newLogger(level LogLevel) *Logger {
	if level == "" {
		level = LogLevelWarn
	}
	return &Logger{level: level}
}

func (l *Logger) Debug(msg string) {
	if l.level == LogLevelDebug {
		log.Println(msg)
	}
}

func (l *Logger) Warn(msg string) {
	if l.level == LogLevelDebug || l.level == LogLevelInfo || l.level == LogLevelWarn {
		log.Println(msg)
	}
}
