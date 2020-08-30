package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
)

func init() {
	log = logrus.New()

	// Use logrus for standard log output
	// Note that `log` here references stdlib's log
	// Not logrus imported under the name `log`.
	log.SetOutput(os.Stdout)
	log.Out = os.Stdout
}

// GlobalFieldsHook is a structure used with logrus to add fixed global field to every log line
type GlobalFieldsHook struct {
	app        string
	appHost    string
	appVersion string
	appPID     int
}

// NewGlobalFieldsHook is a constructor for
func NewGlobalFieldsHook(app, appHost, appVersion string) *GlobalFieldsHook {
	return &GlobalFieldsHook{
		app:        app,
		appHost:    appHost,
		appVersion: appVersion,
		appPID:     os.Getpid(),
	}
}

// Levels implement
func (g *GlobalFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implement
func (g *GlobalFieldsHook) Fire(entry *logrus.Entry) error {
	entry.Data["app"] = g.app
	entry.Data["appHost"] = g.appHost
	entry.Data["appVersion"] = g.appVersion
	entry.Data["appPid"] = g.appPID
	return nil
}

// Debug ...
func Debug(v ...interface{}) {
	log.Debug(v...)
}

// Debugf ...
func Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}

// Infof ...
func Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Info ...
func Info(v ...interface{}) {
	log.Info(v...)
}

// Warnf ...
func Warnf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

// Warn ...
func Warn(v ...interface{}) {
	log.Warn(v...)
}

// Errorf ...
func Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

// Error ...
func Error(v ...interface{}) {
	log.Error(v...)
}

// Fatalf ...
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// Fatal ...
func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

// Panicf ...
func Panicf(format string, v ...interface{}) {
	log.Panicf(format, v...)
}

// Panic ...
func Panic(v ...interface{}) {
	log.Panic(v...)
}

// SetFormatter ...
func SetFormatter(formatter logrus.Formatter) {
	log.SetFormatter(formatter)
}

// SetLevel ...
func SetLevel(level logrus.Level) {
	log.SetLevel(level)
}

// SetReportCaller  ...
func SetReportCaller(reportCaller bool) {
	log.SetReportCaller(reportCaller)
}

// WithFields ...
func WithFields(fields logrus.Fields) *logrus.Entry {
	return log.WithFields(fields)
}

// AddHook ...
func AddHook(hook logrus.Hook) {
	log.AddHook(hook)
}
