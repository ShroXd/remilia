package logger

import (
	"fmt"
	"os"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type logger struct {
	fileWriter   *os.File
	consoleLevel LogLevel
	fileLevel    LogLevel
}

func newLogger(consoleLevel, fileLevel LogLevel, logFile string) (*logger, error) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &logger{
		fileWriter:   f,
		consoleLevel: consoleLevel,
		fileLevel:    fileLevel,
	}, nil
}

func (l *logger) log(level LogLevel, format string, args ...interface{}) {
	if level >= l.consoleLevel {
		fmt.Fprintf(os.Stdout, format, args...)
	}

	if level >= l.fileLevel {
		fmt.Fprintf(l.fileWriter, format, args...)
	}
}

func (l *logger) debug(format string, args ...interface{}) {
	l.log(DEBUG, "DEBUG: "+format, args...)
}

func (l *logger) info(format string, args ...interface{}) {
	l.log(DEBUG, "INFO: "+format, args...)
}

func (l *logger) warn(format string, args ...interface{}) {
	l.log(DEBUG, "WARN: "+format, args...)
}

func (l *logger) error(format string, args ...interface{}) {
	l.log(DEBUG, "ERROR: "+format, args...)
}

func (l *logger) close() error {
	return l.fileWriter.Close()
}

var globalLogger *logger

func New(consoleLevel, fileLevel LogLevel, logFile string) error {
	var err error
	globalLogger, err = newLogger(consoleLevel, fileLevel, logFile)
	return err
}

func Debug(format string, args ...interface{}) {
	globalLogger.debug(format, args...)
}

func Info(format string, args ...interface{}) {
	globalLogger.info(format, args...)
}

func Warn(format string, args ...interface{}) {
	globalLogger.debug(format, args...)
}

func Error(format string, args ...interface{}) {
	globalLogger.debug(format, args...)
}

func Close() error {
	return globalLogger.close()
}
