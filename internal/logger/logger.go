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
	app     string
	host    string
	version string
	pid     int
}

// NewGlobalFieldsHook is a constructor for
func NewGlobalFieldsHook(app, host, version string) *GlobalFieldsHook {
	return &GlobalFieldsHook{
		app:     app,
		host:    host,
		version: version,
		pid:     os.Getpid(),
	}
}

// Levels implement
func (g *GlobalFieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implement
func (g *GlobalFieldsHook) Fire(entry *logrus.Entry) error {
	entry.Data["app"] = g.app
	entry.Data["host"] = g.host
	entry.Data["version"] = g.version
	entry.Data["pid"] = g.pid
	return nil
}

// Infof ...
func Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
}

// Warnf ...
func Warnf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

// Errorf ...
func Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}

// Debugf ...
func Debugf(format string, v ...interface{}) {
	log.Debugf(format, v...)
}
