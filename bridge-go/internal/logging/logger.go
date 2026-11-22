package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	MaxLogSizeMB = 10
	MaxLogFiles  = 2
)

type Logger struct {
	mu    sync.Mutex
	out   io.Writer
	debug bool
	now   func() time.Time
}

func New(out io.Writer, level string) *Logger {
	return &Logger{
		out:   out,
		debug: level == "debug",
		now:   time.Now,
	}
}

func NewRotating(path, level string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close log file: %w", err)
	}
	writer := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    MaxLogSizeMB,
		MaxBackups: MaxLogFiles - 1,
		LocalTime:  true,
	}
	return New(writer, level), nil
}

func (l *Logger) Infof(format string, args ...any) {
	l.write("INFO", format, args...)
}

func (l *Logger) Warningf(format string, args ...any) {
	l.write("WARNING", format, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	if l.debug {
		l.write("DEBUG", format, args...)
	}
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if closer, ok := l.out.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (l *Logger) write(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := fmt.Fprintf(l.out, "%s %-5s %s\n", l.now().Format("2006-01-02 15:04:05"), level, message); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cody-bridge: log write failed: %v\n", err)
	}
}

func (l *Logger) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	line := strings.TrimRight(string(p), "\r\n")
	_, err := fmt.Fprintf(l.out, "%s INFO  %s\n", l.now().Format("2006-01-02 15:04:05"), line)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
