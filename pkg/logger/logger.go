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

func NewLogger(id, name string) (*Logger, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Core for stdout
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
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
		zapcore.DebugLevel,
	)

	core := zapcore.NewTee(consoleCore, fileCore)
	zlogger := zap.New(core)

	scraperLogger := zlogger.With(zap.String("ID", id), zap.String("scraperName", name))

	return &Logger{
		internal: scraperLogger,
	}, nil
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
