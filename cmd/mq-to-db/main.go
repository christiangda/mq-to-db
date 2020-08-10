package main

import (
	"context"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/christiangda/mq-to-db/internal/messages"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/consumer/kafka"
	"github.com/christiangda/mq-to-db/internal/consumer/rmq"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/christiangda/mq-to-db/internal/storage/memory"
	"github.com/christiangda/mq-to-db/internal/storage/pgsql"

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
	conf       config.Config
	log        *logrus.Entry
	rootLogger = logrus.New()
	v          = viper.New()
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
	// Server conf flags
	flag.StringVar(&conf.Server.Address, "address", "127.0.0.1", "Server address")
	flag.IntVar(&conf.Server.Port, "port", 8080, "Server port")
	flag.DurationVar(&conf.Server.ReadTimeout, "readTimeout", 2*time.Second, "Server ReadTimeout")
	flag.DurationVar(&conf.Server.WriteTimeout, "writeTimeout", 5*time.Second, "Server WriteTimeout")
	flag.DurationVar(&conf.Server.IdleTimeout, "idleTimeout", 60*time.Second, "Server IdleTimeout")
	flag.DurationVar(&conf.Server.ReadTimeout, "readHeaderTimeout", 5*time.Second, "Server ReadHeaderTimeout")
	flag.DurationVar(&conf.Server.ShutdownTimeout, "shutdownTimeout", 30*time.Second, "Server ShutdownTimeout")
	flag.BoolVar(&conf.Server.KeepAlivesEnabled, "keepAlivesEnabled", true, "Server KeepAlivesEnabled")
	flag.BoolVar(&conf.Server.Debug, "debug", false, "debug")
	flag.StringVar(&conf.Server.LogFormat, "logFormat", "text", "Log Format [text|json] ")

	// Application conf var
	flag.StringVar(&conf.Application.ConfigFile, "configFile", "config", "Configuration file")

	flag.Parse()

	host, err := os.Hostname()
	if err != nil {
		log.Fatal("Unable to get the host name")
	}

	// Logs conf
	if strings.ToLower(conf.Server.LogFormat) == "json" {
		rootLogger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		rootLogger.SetFormatter(&logrus.TextFormatter{})
	}

	if conf.Server.Debug {
		rootLogger.SetLevel(logrus.DebugLevel)
	} else {
		rootLogger.SetLevel(logrus.InfoLevel)
	}

	rootLogger.SetOutput(os.Stdout)
	// Set the output of the message for the current logrus instance,
	// Output of logrus instance can be set to any io.writer
	rootLogger.Out = os.Stdout

	log = rootLogger.WithFields(logrus.Fields{
		"app":  appName,
		"host": host,
	})

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

	err = v.Unmarshal(&conf)
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
			log.Fatalf("Error creating storage memory: %s", err)
		}
	case "postgresql":
		db, err = pgsql.New(&conf)
		if err != nil {
			log.Fatalf("Error creating storage postgresql: %s", err)
		}
	default:
		log.Panic("Inside configuration file database.kind must be [postgresql|memory]")
	}

	// Select the consumer
	switch conf.Consumer.Kind {
	case "kafka":
		qc, err = kafka.New(&conf)
		if err != nil {
			log.Fatalf("Error creating consumer kafka: %s", err)
		}
	case "rabbitmq":
		qc, err = rmq.New(&conf)
		if err != nil {
			log.Fatalf("Error creating consumer rabbitmq: %s", err)
		}
	default:
		log.Fatal("Inside configuration file consumer.kind must be [rabbitmq|kafka]")
	}

	qc.Connect()
	defer qc.Close()

	// Application context
	appCtx := context.Background()

	if err = db.Ping(appCtx); err != nil {
		log.Fatal("Error conecting to database")
	}
	defer db.Close()

	done := make(chan bool, 1)

	// routine to consume messages
	go func() {
		for m := range qc.Consume() {

			mt, err := messages.GetType(m)
			if err != nil {
				log.Error(err)
			}

			switch mt {
			case "SQL":
				pl := messages.NewSQLMessage(m)
				res, err := db.ExecContext(appCtx, pl.Content.Sentence)
				if err != nil {
					log.Errorf("Error storing SQL payload payload: %v", err)
				}
				log.Debugf("Result %s", res)

			case "RAW":
				log.Warnf("Unknow JSON payload %s", m.Payload)

			default:
				log.Warnf("Unknow payload %s", m.Payload)
			}

		}
	}()

	// main routine blocked until others routines finished
	<-done
}
