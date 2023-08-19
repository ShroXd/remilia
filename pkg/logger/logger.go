package logger

import (
	"log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TODO: add the scraper name to the log
type logger struct {
	internal *zap.Logger
}

func newLogger() (*logger, error) {
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

	return &logger{
		internal: zlogger,
	}, nil
}

var globalLogger *logger

func New() {
	var err error
	globalLogger, err = newLogger()
	if err != nil {
		log.Panic(err)
	}
}

func Debug(msg string, fields ...zap.Field) {
	globalLogger.internal.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	globalLogger.internal.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	globalLogger.internal.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	globalLogger.internal.Error(msg, fields...)
}

func DPanic(msg string, fields ...zap.Field) {
	globalLogger.internal.DPanic(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	globalLogger.internal.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	globalLogger.internal.Fatal(msg, fields...)
}
