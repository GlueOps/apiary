package proxmox

import (
	"fmt"
	"io"
	"os"
)

const (
	LevelError = iota + 1
	LevelWarn
	LevelInfo
	LevelDebug
)

type LeveledLoggerInterface interface {
	Debugf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

type LeveledLogger struct {
	Level int

	stderrOverride io.Writer
	stdoutOverride io.Writer
}

func (l *LeveledLogger) Debugf(format string, v ...interface{}) {
	if l.Level >= LevelDebug {
		_, _ = fmt.Fprintf(l.stdout(), "[DEBUG] "+format+"\n", v...)
	}
}

func (l *LeveledLogger) Errorf(format string, v ...interface{}) {

	if l.Level >= LevelError {
		_, _ = fmt.Fprintf(l.stderr(), "[ERROR] "+format+"\n", v...)
	}
}

func (l *LeveledLogger) Infof(format string, v ...interface{}) {
	if l.Level >= LevelInfo {
		_, _ = fmt.Fprintf(l.stdout(), "[INFO] "+format+"\n", v...)
	}
}

func (l *LeveledLogger) Warnf(format string, v ...interface{}) {
	if l.Level >= LevelWarn {
		_, _ = fmt.Fprintf(l.stderr(), "[WARN] "+format+"\n", v...)
	}
}

func (l *LeveledLogger) stderr() io.Writer {
	if l.stderrOverride != nil {
		return l.stderrOverride
	}

	return os.Stderr
}

func (l *LeveledLogger) stdout() io.Writer {
	if l.stdoutOverride != nil {
		return l.stdoutOverride
	}

	return os.Stdout
}
