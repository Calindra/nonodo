package federationgo

import (
	"log/slog"

	log "github.com/jensneuse/abstractlogger"
)

type MySlog struct {
	instance slog.Logger
}

func NewMySlog() MySlog {
	return MySlog{
		instance: slog.NewLogLogger()
	}
}

// Debug implements abstractlogger.Logger.
func (m MySlog) Debug(msg string, fields ...log.Field) {
	// panic("unimplemented")
	m.instance.Debug(msg, fields)
}

// Error implements abstractlogger.Logger.
func (m MySlog) Error(msg string, fields ...log.Field) {
	panic("unimplemented")
}

// Fatal implements abstractlogger.Logger.
func (m MySlog) Fatal(msg string, fields ...log.Field) {
	panic("unimplemented")
}

// Info implements abstractlogger.Logger.
func (m MySlog) Info(msg string, fields ...log.Field) {
	panic("unimplemented")
}

// LevelLogger implements abstractlogger.Logger.
func (m MySlog) LevelLogger(level log.Level) log.LevelLogger {
	panic("unimplemented")
}

// Panic implements abstractlogger.Logger.
func (m MySlog) Panic(msg string, fields ...log.Field) {
	panic("unimplemented")
}

// Warn implements abstractlogger.Logger.
func (m MySlog) Warn(msg string, fields ...log.Field) {
	panic("unimplemented")
}

func Test() {
	var logger log.Logger = MySlog{}

	logger.Debug("test")
}
