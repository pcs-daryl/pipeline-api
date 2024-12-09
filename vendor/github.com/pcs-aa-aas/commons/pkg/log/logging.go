package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

// init runs the moment this package is imported
func init() {
	SetLogLevel(zapcore.DebugLevel)
}

var logger *zap.SugaredLogger
var logLevel zapcore.Level

// lbjk is a writecloser object that writes the incoming stream to a file
var lbjk = newLogger()

func newLogger() *lumberjack.Logger {
	n := os.Getenv("MICROSERVICE_NAME")
	if n == "" {
		n = "vap"
	}
	return &lumberjack.Logger{
		Filename:   "/vap_logs/" + n + ".log",
		MaxSize:    5, // megabytes
		MaxBackups: 3,
		MaxAge:     10,    //days
		Compress:   false, // disabled by default
	}
}

// lumberjack.Logger is already safe for concurrent use, so we don't need to lock it.
var fileLog = zapcore.AddSync(lbjk)

var consoleDebugging = zapcore.Lock(os.Stdout)
var consoleErrors = zapcore.Lock(os.Stderr)

var encoderConfig = zapcore.EncoderConfig{
	MessageKey:     "m",
	LevelKey:       "l",
	TimeKey:        "t",
	NameKey:        "n",
	CallerKey:      "c",
	StacktraceKey:  "s",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

// TODO: Switch encoder when we do unified logging so that we output json for the log driver
// encoder := zapcore.NewJSONEncoder(encoderConfig)
var encoder = zapcore.NewConsoleEncoder(encoderConfig)

type logConfiguration struct {
	logMinLevel zapcore.Level
	logMaxLevel zapcore.Level
}

// Enabled for conformance to interface. Decides which log level to enable for the logger core.
func (l logConfiguration) Enabled(level zapcore.Level) bool {
	return level >= l.logMinLevel && level <= l.logMaxLevel
}

// SetLogLevel sets the log level
func SetLogLevel(level zapcore.Level) {

	// Join the outputs, encoders, and level-handling functions into
	// zapcore.Cores, then tee the cores together.
	core := zapcore.NewTee(
		//zapcore.NewCore(encoder, fileLog, logConfiguration{logMinLevel: level, logMaxLevel: zapcore.FatalLevel}),
		zapcore.NewCore(encoder, consoleErrors, logConfiguration{logMinLevel: zapcore.ErrorLevel, logMaxLevel: zapcore.FatalLevel}),
		zapcore.NewCore(encoder, consoleDebugging, logConfiguration{logMinLevel: level, logMaxLevel: zapcore.WarnLevel}),
	)

	l := zap.New(core)

	// Uncomment to print caller (who called the log print func) in log
	//l = l.WithOptions(
	//zap.AddCaller(),
	//zap.AddCallerSkip(1),
	//zap.AddStacktrace(zapcore.ErrorLevel),
	//)
	logger = l.Sugar()
	logLevel = level
}

func GetLogLevel() zapcore.Level {
	return logLevel
}

// Debugf logs formatted fine-grained informational events that are most useful for debugging
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Debug logs fine-grained informational events that are most useful for debugging
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Infof logs formatted informational messages that highlight the progress of the app at coarse-grained level
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Info logs informational messages that highlight the progress of the app at coarse-grained level
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warnf logs formatted messages about potentially harmful situations
func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

// Warn logs messages about potentially harmful situations
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Errorf logs formatted messages about error events that might still allow the application to continue running
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// Error logs messages about error events that might still allow the application to continue running
func Error(args ...interface{}) {
	logger.Errorf("%+v", args...)
}

// Deprecated: Print is now a convenience function that logs at Debug level
func Print(args ...interface{}) {
	logger.Debug(args...)
}

// Deprecated: Println is now a convenience function that logs at Debug level
func Println(args ...interface{}) {
	logger.Debug(args...)
}

// Deprecated: Printf is now a convenience function that logs at Debug level
func Printf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

// Fatal logs very severe error events that will presumably lead the app to abort
func Fatal(args ...interface{}) {
	logger.Fatalf("%+v", args...)
}

// Fatalln logs very severe error events that will presumably lead the app to abort
func Fatalln(args ...interface{}) {
	logger.Fatal(args...)
}

// Fatalf logs very severe error events that will presumably lead the app to abort
func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}
