package main

import (
	"path/filepath"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/consumer/kafka"
	"github.com/christiangda/mq-to-db/internal/consumer/rmq"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/christiangda/mq-to-db/internal/storage/memory"
	"github.com/christiangda/mq-to-db/internal/storage/pgsql"

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
	appEnvPrefix     = "MQTODB_"
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
	pflag.StringVarP(&conf.Server.LogFormat, "logFormat", "l", "text", "Log Format [text|json] ")

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

	// TODO: Define default values to be used when config file doesn't has it

	log.Debug(conf.ToYAML())

	// Create abstraction layers (Using interfaces)
	var db storage.Store
	var qc consumer.Consumer

	// Select the storage
	switch conf.Database.Kind {
	case "memory":
		db, err = memory.New(&conf)
		if err != nil {
			log.Error("Error creating storage memory")
		}
	case "postgresql":
		db, err = pgsql.New(&conf)
		if err != nil {
			log.Error("Error creating storage postgresql")
		}
	default:
		log.Panic("Inside configuration file database.kind must be [postgresql|memory]")
	}

	// Select the consumer
	switch conf.Consumer.Kind {
	case "kafka":
		qc, err = kafka.New(&conf)
		if err != nil {
			log.Error("Error creating consumer kafka")
		}
	case "rabbitmq":
		qc, err = rmq.New(&conf)
		if err != nil {
			log.Error("Error creating consumer rabbitmq")
		}
	default:
		log.Fatal("Inside configuration file consumer.kind must be [rabbitmq|kafka]")
	}

	qc.Connect()
	defer qc.Close()

	if err = db.Ping(appCtx); err != nil {
		log.Fatal("Error conecting to database")
	}
	defer db.Close()

	done := make(chan bool, 1)

	// routine to consume messages
	go func() {
		for m := range qc.Consume() {
			log.Debugf("Message Payload: %s", m.Payload)
		}
	}()

	// main routine blocked until others routines finished
	<-done
}
