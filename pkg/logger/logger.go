package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var DefaultLogger = &Logger{
	logger: zap.NewNop().Sugar(),
}

type LogLevel string

const (
	ErrorLevel LogLevel = "error"
	WarnLevel  LogLevel = "warn"
	InfoLevel  LogLevel = "info"
	DebugLevel LogLevel = "debug"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

type Params struct {
	AppName                  string
	LogDir                   string
	Level                    LogLevel
	UseStdAndFile            bool
	AddLowPriorityLevelToCmd bool
}

// Logger -.
type Logger struct {
	logger *zap.SugaredLogger
}

// New -.
func New(param Params, out io.Writer) *Logger {
	core := newZapCore(param, out)

	logger := zap.New(core)

	sugLogger := logger.Sugar()

	return &Logger{
		logger: sugLogger.With(string(AppName), param.AppName),
	}
}

func toZapLevel(level LogLevel) zapcore.Level {
	switch LogLevel(strings.ToLower(string(level))) {
	case ErrorLevel:
		return zap.ErrorLevel
	case WarnLevel:
		return zap.WarnLevel
	case InfoLevel:
		return zap.InfoLevel
	case DebugLevel:
		return zap.DebugLevel
	case PanicLevel:
		return zap.PanicLevel
	case FatalLevel:
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

func newZapCore(param Params, out io.Writer) (core zapcore.Core) {
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= toZapLevel(param.Level)
	})

	if param.AddLowPriorityLevelToCmd { // separate levels
		core = withLowePriorityLevel(param, out, highPriority)
	} else { // not separate levels
		core = withoutLowePriorityLevel(param, out, highPriority)
	}

	return core
}

func withoutLowePriorityLevel(param Params, out io.Writer, highPriority zap.LevelEnablerFunc) zapcore.Core {
	topicErrors := zapcore.AddSync(out)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	if param.UseStdAndFile && param.LogDir != "" {
		consoleErrors := zapcore.Lock(os.Stderr)
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

		return zapcore.NewTee(
			zapcore.NewCore(fileEncoder, topicErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		)
	}

	return zapcore.NewTee(
		zapcore.NewCore(fileEncoder, topicErrors, highPriority),
	)
}

func withLowePriorityLevel(param Params, out io.Writer, highPriority zap.LevelEnablerFunc) zapcore.Core {
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < toZapLevel(param.Level)
	})

	topicDebugging := zapcore.AddSync(out)
	topicErrors := zapcore.AddSync(out)
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	if param.UseStdAndFile && param.LogDir != "" {
		consoleDebugging := zapcore.Lock(os.Stdout)
		consoleErrors := zapcore.Lock(os.Stderr)
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

		return zapcore.NewTee(
			zapcore.NewCore(fileEncoder, topicErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(fileEncoder, topicDebugging, lowPriority),
			zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		)
	}

	return zapcore.NewTee(
		zapcore.NewCore(fileEncoder, topicErrors, highPriority),
		zapcore.NewCore(fileEncoder, topicDebugging, lowPriority),
	)
}

func (l *Logger) Sync() error {
	return l.logger.Sync()
}

// Debug -.
func (l *Logger) Debug(message any, args ...any) {
	l.log(l.logger.Debugf, message, args...)
}

// Info -.
func (l *Logger) Info(message any, args ...any) {
	l.log(l.logger.Infof, message, args...)
}

// Warn -.
func (l *Logger) Warn(message any, args ...any) {
	l.log(l.logger.Warnf, message, args...)
}

// Panic -.
func (l *Logger) Panic(message any, args ...any) {
	l.log(l.logger.Panicf, message, args...)
}

// Error -.
func (l *Logger) Error(message any, args ...any) {
	l.log(l.logger.Errorf, message, args...)
}

// Fatal -.
func (l *Logger) Fatal(message any, args ...any) {
	l.log(l.logger.Fatalf, message, args...)
}

func (*Logger) log(lg func(message string, args ...any), message any, args ...any) {
	switch tp := message.(type) {
	case error:
		lg(tp.Error(), args...)
	case string:
		lg(tp, args...)
	default:
		lg(fmt.Sprintf("message %v has unknown type %v", message, tp), args...)
	}
}

func (l *Logger) With(key Field, value any) Interface {
	return &Logger{l.logger.With(string(key), value)}
}
