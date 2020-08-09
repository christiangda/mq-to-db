package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"

	"os"
	"strings"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	appName          = "mq-to-db"
	appDescription   = "This is a Message Queue Consumer to store the payload into Database"
	appGitRepository = "https://github.com/christiangda/mq-to-db"
	appMetricsPath   = "/metrics"
	appHealthPath    = "/health"
	appEnvPrefix     = "RMQTOPGQL_"
)

var (
	conf config.Config
	log  = logrus.New()
	v    = viper.New()
)

func init() {

	// Set default values
	conf.Application.Name = appName
	conf.Application.Description = appDescription
	conf.Application.GitRepository = appGitRepository
	conf.Application.MetricsPath = appMetricsPath
	conf.Application.HealthPath = appHealthPath
	conf.Application.Version = version.Version
	conf.Application.Revision = version.Revision
	conf.Application.GoVersion = version.GoVersion
	conf.Application.BuildUser = version.BuildUser
	conf.Application.BuildDate = version.BuildDate
	conf.Application.VersionInfo = version.Info()
	conf.Application.BuildInfo = version.BuildContext()
}

func main() {
	// cmd flags
	pflag.StringVarP(&conf.Application.ConfigFile, "configFile", "c", "config", "Configuration file")
	pflag.BoolVarP(&conf.Server.Debug, "debug", "d", false, "debug")
	pflag.StringVarP(&conf.Server.LogFormat, "logFormat", "l", "json", "Log Format")

	pflag.Parse()

	// Logs conf
	if strings.ToLower(conf.Server.LogFormat) == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{})
	}

	if conf.Server.Debug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.SetOutput(os.Stdout)

	// Set the output of the message for the current logrus instance,
	// Output of logrus instance can be set to any io.writer
	log.Out = os.Stdout

	// Read config file
	v.SetConfigType("yaml")
	v.SetConfigName(filepath.Base(conf.Application.ConfigFile))
	v.AddConfigPath(filepath.Dir(conf.Application.ConfigFile))
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/" + appName + "/")
	v.AddConfigPath("$HOME/." + appName)

	// Env Vars
	v.SetEnvPrefix(appEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	log.Debugf("Available Env Vars: %s", os.Environ())

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	err := v.Unmarshal(&conf)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	fmt.Println(conf.ToYAML())
}
