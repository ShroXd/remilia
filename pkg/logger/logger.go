package logger

import (
	"log"

	"go.uber.org/zap"
)

type logger struct {
	internal *zap.Logger
}

func newLogger() (*logger, error) {
	zlogger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

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
