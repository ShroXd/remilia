package remilia

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logContext map[string]interface{}

type Logger interface {
	Debug(msg string, context ...logContext)
	Info(msg string, context ...logContext)
	Warn(msg string, context ...logContext)
	Error(msg string, context ...logContext)
	Panic(msg string, context ...logContext)
}

type defaultLogger struct {
	internal *zap.Logger
}

func (l *defaultLogger) Debug(msg string, context ...logContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Debug(msg, fields...)
}

func (l *defaultLogger) Info(msg string, context ...logContext) {
	fields := convertToZapFields(getContext((context)))
	l.internal.Info(msg, fields...)
}

func (l *defaultLogger) Warn(msg string, context ...logContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Warn(msg, fields...)
}

func (l *defaultLogger) Error(msg string, context ...logContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Error(msg, fields...)
}

func (l *defaultLogger) Panic(msg string, context ...logContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Panic(msg, fields...)
}

func getContext(context []logContext) logContext {
	if len(context) > 0 {
		return context[0]
	}

	return nil
}

type logLevel int8

const (
	debugLevel logLevel = iota - 1
	infoLevel
	warnLevel
	errorLevel
)

func (level logLevel) toZapLevel() zapcore.Level {
	switch level {
	case debugLevel:
		return zap.DebugLevel
	case infoLevel:
		return zap.InfoLevel
	case warnLevel:
		return zap.WarnLevel
	case errorLevel:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

type loggerConfig struct {
	ID           string
	Name         string
	ConsoleLevel logLevel
	FileLevel    logLevel
}

func newConsoleCore(encoderConfig zapcore.EncoderConfig, level zapcore.Level) zapcore.Core {
	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.Lock(os.Stdout),
		level,
	)
}

func newFileCore(fs fileSystemOperations, encoderConfig zapcore.EncoderConfig, level zapcore.Level, fileName string) (zapcore.Core, error) {
	logDir := "logs"
	if err := fs.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, err
	}

	logFilePath := filepath.Join(logDir, fileName)
	file, err := fs.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		level,
	), nil
}

func getLogFileName(c *loggerConfig) string {
	timeFormat := "20060102_150405"
	return fmt.Sprintf("%s_%s_%s.log", c.ID, c.Name, time.Now().Format(timeFormat))
}

// TODO: refactor to functional options so that user can set the log path etc..
func createLogger(c *loggerConfig, fs fileSystemOperations) (*defaultLogger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleCore := newConsoleCore(encoderConfig, c.ConsoleLevel.toZapLevel())
	fileCore, err := newFileCore(fs, encoderConfig, c.FileLevel.toZapLevel(), getLogFileName(c))
	if err != nil {
		return nil, err
	}

	core := zapcore.NewTee(consoleCore, fileCore)
	zlogger := zap.New(core).With(
		zap.String("ID", c.ID),
		zap.String("Name", c.Name),
	)

	return &defaultLogger{
		internal: zlogger,
	}, nil
}

func convertToZapFields(context map[string]interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(context))
	for k, v := range context {
		fields = append(fields, zap.Any(k, v))
	}

	return fields
}
