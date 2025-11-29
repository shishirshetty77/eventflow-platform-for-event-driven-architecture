// Package logging provides structured logging utilities using zap.
// It wraps zap to provide consistent logging across all microservices.
package logging

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is used for context value keys.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
	// TraceIDKey is the context key for trace ID.
	TraceIDKey contextKey = "trace_id"
	// SpanIDKey is the context key for span ID.
	SpanIDKey contextKey = "span_id"
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "user_id"
)

// Logger wraps zap.Logger with additional functionality.
type Logger struct {
	*zap.Logger
	serviceName string
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Config represents logger configuration.
type Config struct {
	Level       string   `json:"level"`
	Development bool     `json:"development"`
	ServiceName string   `json:"service_name"`
	OutputPaths []string `json:"output_paths"`
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig(serviceName string) *Config {
	return &Config{
		Level:       "info",
		Development: false,
		ServiceName: serviceName,
		OutputPaths: []string{"stdout"},
	}
}

// NewLogger creates a new Logger instance with the given configuration.
func NewLogger(cfg *Config) (*Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Development {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Build output writers
	var writers []zapcore.WriteSyncer
	for _, path := range cfg.OutputPaths {
		switch path {
		case "stdout":
			writers = append(writers, zapcore.AddSync(os.Stdout))
		case "stderr":
			writers = append(writers, zapcore.AddSync(os.Stderr))
		default:
			file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, err
			}
			writers = append(writers, zapcore.AddSync(file))
		}
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// Add service name field
	zapLogger = zapLogger.With(zap.String("service", cfg.ServiceName))

	return &Logger{
		Logger:      zapLogger,
		serviceName: cfg.ServiceName,
	}, nil
}

// Init initializes the default logger with the given configuration.
func Init(cfg *Config) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(cfg)
	})
	return err
}

// Default returns the default logger instance.
// If not initialized, it creates a development logger.
func Default() *Logger {
	if defaultLogger == nil {
		cfg := DefaultConfig("default")
		cfg.Development = true
		logger, _ := NewLogger(cfg)
		defaultLogger = logger
	}
	return defaultLogger
}

// WithContext returns a logger with context fields.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := make([]zap.Field, 0, 4)

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	return &Logger{
		Logger:      l.Logger.With(fields...),
		serviceName: l.serviceName,
	}
}

// WithFields returns a logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{
		Logger:      l.Logger.With(zapFields...),
		serviceName: l.serviceName,
	}
}

// WithError returns a logger with an error field.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.Error(err)),
		serviceName: l.serviceName,
	}
}

// ServiceName returns the service name associated with this logger.
func (l *Logger) ServiceName() string {
	return l.serviceName
}

// Field creates a zap.Field for use with the logger.
func Field(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// String creates a string field.
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int creates an int field.
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Int64 creates an int64 field.
func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

// Float64 creates a float64 field.
func Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

// Bool creates a bool field.
func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Error creates an error field.
func Error(err error) zap.Field {
	return zap.Error(err)
}

// Contextual logging helpers

// InfoCtx logs an info message with context.
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Default().WithContext(ctx).Info(msg, fields...)
}

// ErrorCtx logs an error message with context.
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Default().WithContext(ctx).Error(msg, fields...)
}

// WarnCtx logs a warning message with context.
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Default().WithContext(ctx).Warn(msg, fields...)
}

// DebugCtx logs a debug message with context.
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Default().WithContext(ctx).Debug(msg, fields...)
}
