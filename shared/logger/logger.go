package logger

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志接口
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
}

// Config 日志配置
type Config struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`         // json, text
	Output     string `json:"output" yaml:"output"`         // stdout, file
	FilePath   string `json:"file_path" yaml:"file_path"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`     // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"`       // 天
	Compress   bool   `json:"compress" yaml:"compress"`
}

// zapLogger Zap日志实现
type zapLogger struct {
	logger *zap.SugaredLogger
}

// logrusLogger Logrus日志实现
type logrusLogger struct {
	logger *logrus.Entry
}

// NewZapLogger 创建Zap日志实例
func NewZapLogger(config Config) (Logger, error) {
	level := parseLogLevel(config.Level)
	
	// 配置编码器
	var encoderConfig zapcore.EncoderConfig
	if config.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.LevelKey = "level"
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.CallerKey = "caller"
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	
	// 创建编码器
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}
	
	// 配置输出
	var writeSyncer zapcore.WriteSyncer
	if config.Output == "file" && config.FilePath != "" {
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
		writeSyncer = zapcore.AddSync(file)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}
	
	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)
	
	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	
	return &zapLogger{
		logger: logger.Sugar(),
	}, nil
}

// NewLogrusLogger 创建Logrus日志实例
func NewLogrusLogger(config Config) (Logger, error) {
	logger := logrus.New()
	
	// 设置日志级别
	level := parseLogrusLevel(config.Level)
	logger.SetLevel(level)
	
	// 设置日志格式
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}
	
	// 设置输出
	if config.Output == "file" && config.FilePath != "" {
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
		logger.SetOutput(file)
	} else {
		logger.SetOutput(os.Stdout)
	}
	
	return &logrusLogger{
		logger: logrus.NewEntry(logger),
	}, nil
}

// Zap Logger 实现
func (l *zapLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *zapLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *zapLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *zapLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *zapLogger) WithField(key string, value interface{}) Logger {
	return &zapLogger{
		logger: l.logger.With(key, value),
	}
}

func (l *zapLogger) WithFields(fields map[string]interface{}) Logger {
	var args []interface{}
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &zapLogger{
		logger: l.logger.With(args...),
	}
}

func (l *zapLogger) WithContext(ctx context.Context) Logger {
	// 从上下文中提取追踪信息
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		return l.WithField("trace_id", traceID)
	}
	return l
}

// Logrus Logger 实现
func (l *logrusLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *logrusLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *logrusLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *logrusLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *logrusLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *logrusLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		logger: l.logger.WithField(key, value),
	}
}

func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &logrusLogger{
		logger: l.logger.WithFields(fields),
	}
}

func (l *logrusLogger) WithContext(ctx context.Context) Logger {
	// 从上下文中提取追踪信息
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		return l.WithField("trace_id", traceID)
	}
	return l
}

// 辅助函数
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func parseLogrusLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}

func getTraceIDFromContext(ctx context.Context) string {
	// 这里可以集成OpenTelemetry来获取trace ID
	// 暂时返回空字符串
	return ""
}

// 全局默认logger
var defaultLogger Logger

// SetDefault 设置默认logger
func SetDefault(logger Logger) {
	defaultLogger = logger
}

// 全局便捷方法
func Debug(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(args...)
	}
}

func Info(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(args...)
	}
}

func Warn(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(args...)
	}
}

func Error(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(args...)
	}
}

func Fatal(args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debugf(format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Infof(format, args...)
	}
}

func Warnf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warnf(format, args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Errorf(format, args...)
	}
}

func Fatalf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatalf(format, args...)
	}
}