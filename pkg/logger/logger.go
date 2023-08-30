package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TODO: add the scraper name to the log
type Logger struct {
	internal *zap.Logger
}

type LoggerConfig struct {
	ID              string
	Name            string
	ConsoleLogLevel LogLevel
	FileLogLevel    LogLevel
}

type LogLevel int8

const (
	DebugLevel LogLevel = iota - 1
	InfoLevel
	WarnLevel
	ErrorLevel
	DPanicLevel
	PanicLevel
	FatalLevel
)

func NewLogger(config *LoggerConfig) (*Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Core for stdout
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		config.ConsoleLogLevel.ToZapLevel(),
	)

	currDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	logDir := filepath.Join(currDir, "logs")
	err = os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// Core for log file
	logFilePath := filepath.Join(logDir, "logfile.log")
	file, err := os.Create(logFilePath)
	if err != nil {
		return nil, err
	}
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		config.FileLogLevel.ToZapLevel(),
	)

	core := zapcore.NewTee(consoleCore, fileCore)
	zlogger := zap.New(core)

	scraperLogger := zlogger.With(zap.String("ID", config.ID), zap.String("scraperName", config.Name))

	return &Logger{
		internal: scraperLogger,
	}, nil
}

func (level LogLevel) ToZapLevel() zapcore.Level {
	switch level {
	case DebugLevel:
		return zap.DebugLevel
	case InfoLevel:
		return zap.InfoLevel
	case WarnLevel:
		return zap.WarnLevel
	case ErrorLevel:
		return zap.ErrorLevel
	case DPanicLevel:
		return zap.DPanicLevel
	case PanicLevel:
		return zap.PanicLevel
	case FatalLevel:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func (level LogLevel) ToString() string {
	switch level {
	case DebugLevel:
		return "Debug"
	case InfoLevel:
		return "Info"
	case WarnLevel:
		return "Warn"
	case ErrorLevel:
		return "Error"
	case DPanicLevel:
		return "DPanic"
	case PanicLevel:
		return "Panic"
	case FatalLevel:
		return "Fatal"
	default:
		return "Unknown"
	}
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.internal.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.internal.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.internal.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.internal.Error(msg, fields...)
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.internal.DPanic(msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.internal.Panic(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.internal.Fatal(msg, fields...)
}
