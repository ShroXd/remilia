package remilia

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerLevels(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	zapLogger := zap.New(core)
	logger := &DefaultLogger{internal: zapLogger}

	tests := []struct {
		name    string
		logFunc func(msg string, context ...LogContext)
		level   zapcore.Level
	}{
		{"Debug", logger.Debug, zap.DebugLevel},
		{"Info", logger.Info, zap.InfoLevel},
		{"Warn", logger.Warn, zap.WarnLevel},
		{"Error", logger.Error, zap.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorded.TakeAll()
			context := LogContext{"key": "value"}

			tt.logFunc("test message", context)

			entries := recorded.All()
			assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
			entry := entries[0]

			assert.Equal(t, tt.level, entry.Level, "Incorrect log level")
			assert.Equal(t, "test message", entry.Message, "Incorrect message")
			assert.Equal(t, "value", entry.ContextMap()["key"], "Incorrect context logged")
		})
	}

	t.Run("no context", func(t *testing.T) {
		recorded.TakeAll()

		logger.Info("test message")

		entries := recorded.All()
		assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
		entry := entries[0]

		assert.Equal(t, zap.InfoLevel, entry.Level, "Incorrect log level")
		assert.Equal(t, "test message", entry.Message, "Incorrect message")
		assert.Empty(t, entry.ContextMap(), "Expected no context to be logged")
	})
}

func TestPanicLog(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	zapLogger := zap.New(core)
	logger := &DefaultLogger{internal: zapLogger}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic to be logged")
		}

		entries := recorded.All()
		assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
		entry := entries[0]

		assert.Equal(t, zap.PanicLevel, entry.Level, "Incorrect log level")
		assert.Equal(t, "test message", entry.Message, "Incorrect message")
	}()

	logger.Panic("test message", LogContext{"key": "value"})
}

func TestToZapLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
		expected zapcore.Level
	}{
		{"DebugLevel", DebugLevel, zapcore.DebugLevel},
		{"InfoLevel", InfoLevel, zapcore.InfoLevel},
		{"WarnLevel", WarnLevel, zapcore.WarnLevel},
		{"ErrorLevel", ErrorLevel, zapcore.ErrorLevel},
		{"Default", 127, zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.logLevel.toZapLevel()
			assert.Equal(t, tt.expected, actual, "Incorrect zap level")
		})
	}
}

func TestNewConsoleCore(t *testing.T) {
	tests := []struct {
		name        string
		level       zapcore.Level
		expectLevel zapcore.Level
	}{
		{
			name:        "Debug Level",
			level:       zapcore.DebugLevel,
			expectLevel: zapcore.DebugLevel,
		},
		{
			name:        "Info Level",
			level:       zapcore.InfoLevel,
			expectLevel: zapcore.InfoLevel,
		},
		{
			name:        "Warn Level",
			level:       zapcore.WarnLevel,
			expectLevel: zapcore.WarnLevel,
		},
		{
			name:        "Error Level",
			level:       zapcore.ErrorLevel,
			expectLevel: zapcore.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoderConfig := zap.NewProductionEncoderConfig()
			core := newConsoleCore(encoderConfig, tt.level)

			assert.NotNil(t, core, "Expected core to be created")
			assert.True(t, core.Enabled(tt.level), "Expected core to be enabled")
		})
	}
}

func TestNewFileCore(t *testing.T) {
	tests := []struct {
		name         string
		level        zapcore.Level
		expectExists bool
	}{
		{
			name:         "Debug Level",
			level:        zapcore.DebugLevel,
			expectExists: true,
		},
		{
			name:         "Info Level",
			level:        zapcore.InfoLevel,
			expectExists: true,
		},
		{
			name:         "Warn Level",
			level:        zapcore.WarnLevel,
			expectExists: true,
		},
		{
			name:         "Error Level",
			level:        zapcore.ErrorLevel,
			expectExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: consider if it's necessary to use MockFileSystem
			fs := &FileSystem{}
			encoderConfig := zap.NewProductionEncoderConfig()
			logFileName := getLogFileName(&LoggerConfig{
				ID:   "test",
				Name: "unit",
			})

			core, err := newFileCore(fs, encoderConfig, tt.level, logFileName)
			defer os.Remove(filepath.Join("logs", logFileName))

			assert.NotNil(t, core, "Expected core to be created")
			assert.NoError(t, err, "Expected no error")

			_, err = os.Stat(filepath.Join("logs", logFileName))
			assert.Equal(t, tt.expectExists, !os.IsNotExist(err), "Expected log file to exist")
		})
	}

	t.Run("Error on creating log file", func(t *testing.T) {
		mockFS := MockFileSystem{
			MkdirAllErr: errors.New("mkdir error"),
		}

		encoderConfig := zap.NewProductionEncoderConfig()
		logFileName := getLogFileName(&LoggerConfig{
			ID:   "test",
			Name: "unit",
		})

		core, err := newFileCore(mockFS, encoderConfig, zapcore.DebugLevel, logFileName)
		defer os.Remove(filepath.Join("logs", logFileName))

		assert.Nil(t, core, "Expected core to be nil")
		assert.Error(t, err, "Expected error")
		assert.Equal(t, "mkdir error", err.Error(), "Incorrect error message")
	})

	t.Run("Error on open file", func(t *testing.T) {
		mockFS := MockFileSystem{
			OpenFileErr: errors.New("test open file error"),
		}

		encoderConfig := zap.NewProductionEncoderConfig()
		logFileName := getLogFileName(&LoggerConfig{
			ID:   "test",
			Name: "unit",
		})

		core, err := newFileCore(mockFS, encoderConfig, zapcore.DebugLevel, logFileName)
		defer os.Remove(filepath.Join("logs", logFileName))

		assert.Nil(t, core, "Expected core to be nil")
		assert.Error(t, err, "Expected error")
		assert.Equal(t, "test open file error", err.Error(), "Incorrect error message")
	})
}

func TestGetLogFileName(t *testing.T) {
	testConfig := LoggerConfig{
		ID:   "123",
		Name: "testLogger",
	}

	fileName := getLogFileName(&testConfig)

	// Pattern: [ID]_[Name]_[YYYYMMDD_HHMMSS].log
	regexPattern := fmt.Sprintf(`^%s_%s_\d{8}_\d{6}\.log$`, testConfig.ID, testConfig.Name)
	fileNameRegex := regexp.MustCompile(regexPattern)

	assert.Regexp(t, fileNameRegex, fileName, "Incorrect file name")
}

func TestCreateLogger(t *testing.T) {
	tests := []struct {
		name      string
		config    LoggerConfig
		expectErr bool
	}{
		{
			name: "Valid Config",
			config: LoggerConfig{
				ID:           "testID",
				Name:         "testLogger",
				ConsoleLevel: InfoLevel,
				FileLevel:    DebugLevel,
			},
			expectErr: false,
		},
	}

	fs := &FileSystem{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := createLogger(&tt.config, fs)

			if tt.expectErr {
				assert.Error(t, err, "Expected error")
			} else {
				assert.NoError(t, err, "Expected no error")
				assert.NotNil(t, logger, "Expected logger to be created")
				assert.NotNil(t, logger.internal, "Expected internal logger to be created")
			}
		})
	}

	t.Run("Error on creating log file", func(t *testing.T) {
		mockFS := MockFileSystem{
			MkdirAllErr: errors.New("test mkdir error"),
		}

		config := LoggerConfig{
			ID:           "testID",
			Name:         "testLogger",
			ConsoleLevel: InfoLevel,
			FileLevel:    DebugLevel,
		}

		logger, err := createLogger(&config, mockFS)

		assert.Nil(t, logger, "Expected logger to be nil")
		assert.Error(t, err, "Expected error")
		assert.Equal(t, "test mkdir error", err.Error(), "Incorrect error message")
	})
}

func TestConvertToZapFields(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]zap.Field
	}{
		{
			name:     "Empty Map",
			input:    map[string]interface{}{},
			expected: map[string]zap.Field{},
		},
		{
			name: "Single Field",
			input: map[string]interface{}{
				"key1": "value1",
			},
			expected: map[string]zap.Field{
				"key1": zap.Any("key1", "value1"),
			},
		},
		{
			name: "Multiple Fields",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			expected: map[string]zap.Field{
				"key1": zap.Any("key1", "value1"),
				"key2": zap.Any("key2", 123),
				"key3": zap.Any("key3", true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToZapFields(tt.input)

			assert.Len(t, result, len(tt.expected), "Incorrect number of fields")

			for _, field := range result {
				assert.Equal(t, tt.expected[field.Key], field, "Incorrect field")
			}
		})
	}
}
