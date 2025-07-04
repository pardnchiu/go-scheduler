package cron

import (
	"fmt"
	"io"
	"log"
	"os"
)

func (l *StandardLogger) Info(format string, args ...interface{}) {
	l.Printf("[INFO] "+format, args...)
}

func (l *StandardLogger) Error(format string, args ...interface{}) {
	l.Printf("[ERROR] "+format, args...)
}

func (l *StandardLogger) Debug(format string, args ...interface{}) {
	l.Printf("[DEBUG] "+format, args...)
}

type WriterLogger struct {
	writer io.Writer
}

func (l *WriterLogger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

func (l *WriterLogger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

func (l *WriterLogger) Debug(format string, args ...interface{}) {
	l.log("DEBUG", format, args...)
}

func (l *WriterLogger) log(level, format string, args ...interface{}) {
	message := fmt.Sprintf("[%s] %s\n", level, fmt.Sprintf(format, args...))
	l.writer.Write([]byte(message))
}

func (l *NoOpLogger) Info(format string, args ...interface{})  {}
func (l *NoOpLogger) Error(format string, args ...interface{}) {}
func (l *NoOpLogger) Debug(format string, args ...interface{}) {}

func NewLoggerFromStdLogger(stdLogger *log.Logger) Logger {
	return &StandardLogger{Logger: stdLogger}
}

func NewLoggerFromWriter(writer io.Writer) Logger {
	return &WriterLogger{writer: writer}
}

func NewLogger() Logger {
	return NewLoggerFromWriter(os.Stderr)
}
