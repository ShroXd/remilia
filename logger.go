package remilia

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogContext map[string]interface{}
type Logger interface {
	Debug(msg string, context ...LogContext)
	Info(msg string, context ...LogContext)
	Warn(msg string, context ...LogContext)
	Error(msg string, context ...LogContext)
	Panic(msg string, context ...LogContext)
	Fatal(msg string, context ...LogContext)
}

type DefaultLogger struct {
	internal *zap.Logger
}

func (l *DefaultLogger) Debug(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Debug(msg, fields...)
}

func (l *DefaultLogger) Info(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext((context)))
	l.internal.Info(msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Warn(msg, fields...)
}

func (l *DefaultLogger) Error(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Error(msg, fields...)
}

func (l *DefaultLogger) Panic(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Panic(msg, fields...)
}

func (l *DefaultLogger) Fatal(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Fatal(msg, fields...)
}

func getContext(context []LogContext) LogContext {
	if len(context) > 0 {
		return context[0]
	}

	return nil
}

type LogLevel int8

const (
	DebugLevel LogLevel = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (level LogLevel) toZapLevel() zapcore.Level {
	switch level {
	case DebugLevel:
		return zap.DebugLevel
	case InfoLevel:
		return zap.InfoLevel
	case WarnLevel:
		return zap.WarnLevel
	case ErrorLevel:
		return zap.ErrorLevel
	case FatalLevel:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

type LoggerConfig struct {
	ID           string
	Name         string
	ConsoleLevel LogLevel
	FileLevel    LogLevel
}

func newConsoleCore(encoderConfig zapcore.EncoderConfig, level zapcore.Level) zapcore.Core {
	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.Lock(os.Stdout), // using Lock for concurrent safety
		level,
	)
}

func newFileCore(encoderConfig zapcore.EncoderConfig, level zapcore.Level) (zapcore.Core, error) {
	logDir := "logs" // Assuming logs directory is at the same level as the executable
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, err
	}

	logFilePath := filepath.Join(logDir, "logfile.log")
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		level,
	), nil
}

func createLogger(c *LoggerConfig) (*DefaultLogger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	consoleCore := newConsoleCore(encoderConfig, c.ConsoleLevel.toZapLevel())
	fileCore, err := newFileCore(encoderConfig, c.FileLevel.toZapLevel())
	if err != nil {
		return nil, err
	}

	core := zapcore.NewTee(consoleCore, fileCore)
	zlogger := zap.New(core).With(
		zap.String("ID", c.ID),
		zap.String("Name", c.Name),
	)

	return &DefaultLogger{
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
