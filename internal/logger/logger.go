package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jesusjhoel/beam/internal/config"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	project string
	level   Level
	out     io.Writer
	file    *os.File
}

func New(project string, level Level) (*Logger, error) {
	logsDir := config.LogsDir()
	if err := os.MkdirAll(logsDir, 0o700); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}
	logPath := filepath.Join(logsDir, project+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return &Logger{
		project: project,
		level:   level,
		out:     io.MultiWriter(os.Stderr, f),
		file:    f,
	}, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) log(level, msg string) {
	fmt.Fprintf(l.out, "%s [%s] %s: %s\n",
		time.Now().Format("15:04:05"), level, l.project, msg)
}

func (l *Logger) Info(msg string)  { l.log("INFO ", msg) }
func (l *Logger) Warn(msg string)  { l.log("WARN ", msg) }
func (l *Logger) Error(msg string) { l.log("ERROR", msg) }
func (l *Logger) Debug(msg string) {
	if l.level <= LevelDebug {
		l.log("DEBUG", msg)
	}
}

func (l *Logger) Infof(format string, args ...any)  { l.Info(fmt.Sprintf(format, args...)) }
func (l *Logger) Warnf(format string, args ...any)  { l.Warn(fmt.Sprintf(format, args...)) }
func (l *Logger) Errorf(format string, args ...any) { l.Error(fmt.Sprintf(format, args...)) }
func (l *Logger) Debugf(format string, args ...any) { l.Debug(fmt.Sprintf(format, args...)) }
