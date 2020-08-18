package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
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

	log        *logrus.Entry
	rootLogger = logrus.New()
	v          = viper.New()
)

func init() { //package initializer

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
	conf.Application.VersionInfo = version.GetVersionInfo()
	conf.Application.BuildInfo = version.GetVersionInfoExtended()

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
	// Application version
	showVersion := flag.Bool("version", false, "Show application version")
	showVersionInfo := flag.Bool("versionInfo", false, "Show application version information")
	showBuildInfo := flag.Bool("buildInfo", false, "Show application build information")

	flag.Parse()

	if *showVersion {
		fmt.Println(conf.Application.Version)
		os.Exit(0)
	}

	if *showVersionInfo {
		fmt.Println(conf.Application.VersionInfo)
		os.Exit(0)
	}

	if *showBuildInfo {
		fmt.Println(conf.Application.BuildInfo)
		os.Exit(0)
	}
	// Use logrus for standard log output
	// Note that `log` here references stdlib's log
	// Not logrus imported under the name `log`.
	rootLogger.SetOutput(os.Stdout)

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

	host, _ := os.Hostname()
	log = rootLogger.WithFields(logrus.Fields{
		"app":  appName,
		"host": host,
	})

	log.Info("Application initialized")

}

func main() {

	log.Infof("Starting...")

	// Viper default values to conf parameters when config file doesn't have it
	// The config file values overrides these
	// ***** DATABASE *****
	// database.kind: postgresql
	// database.port: 5432
	// database.sslMode: disable
	// database.maxPingTimeOut: 1s
	// database.maxQueryTimeOut: 30s
	// database.connMaxLifetime: 0
	// database.maxIdleConns: 5
	// database.maxOpenConns: 20
	v.SetDefault("database.kind", "postgresql")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslMode", "disable")
	v.SetDefault("Database.maxPingTimeOut", "1s")
	v.SetDefault("Database.maxQueryTimeOut", "30s")
	v.SetDefault("Database.connMaxLifetime", 0)
	v.SetDefault("Database.maxIdleConns", 5)
	v.SetDefault("Database.maxIdleConns", 20)
	// ***** RabbitMQ *****
	// consumer.kind: rabbitmq
	// consumer.port: 5672
	// consumer.requestedHeartbeat: 25
	// consumer.queue.autoACK: false
	v.SetDefault("consumer.kind", "rabbitmq")
	v.SetDefault("consumer.port", 5672)
	v.SetDefault("consumer.requestedHeartbeat", "10s")
	v.SetDefault("consumer.queue.exclusive", false)
	v.SetDefault("consumer.queue.autoACK", false)

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

	log.Debugf("Environment Variables: %s", os.Environ())

	log.Infof("Loading configuration file: %s", conf.Application.ConfigFile)
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	log.Info("Configuring application")
	if err := v.Unmarshal(&conf); err != nil {
		log.Fatalf("Unable to decode: %v", err)
	}

	log.Debugf("Application configuration: %s", conf.ToJSON())

	// Create abstraction layers (Using interfaces)
	var db storage.Store
	var qc consumer.Consumer
	var err error // Necessary to handle errors inside switch/case

	// Select the storage
	switch conf.Database.Kind {
	case "memory":
		db, err = memory.New(&conf)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using  memory database")
	case "postgresql":
		db, err = pgsql.New(&conf)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using postgresql database")
	default:
		log.Panic("Inside configuration file database.kind must be [postgresql|memory]")
	}

	// Select the consumer
	switch conf.Consumer.Kind {
	case "kafka":
		qc, err = kafka.New(&conf)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using kafka consumer")
	case "rabbitmq":
		qc, err = rmq.New(&conf)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using rabbitmq consumer")
	default:
		log.Fatal("Inside configuration file consumer.kind must be [rabbitmq|kafka]")
	}

	// Application context
	appCtx := context.Background()

	// Try to connecto to Storage first, and if everithing is ready, then go for Consumer
	log.Infof("Connecting to database")
	if err := db.Connect(appCtx); err != nil {
		log.Fatal("Error conecting to database")
	}
	defer db.Close() // is here for a moment

	// Try to connect to queue consumer
	log.Infof("Connecting to consumer")
	qc.Connect()
	defer qc.Close() // is here for a moment

	// This channel is used to wait all go routines
	done := make(chan bool, 1)

	// Consume message
	iter, err := qc.Consume() // Consumming messages from RabbitMQ channel
	if err != nil {
		log.Error(err)
	}

	go func(done *chan bool) { // This go routine is to consume message from iterator

		qcm, err := iter.Next()
		if err != nil {
			log.Errorf("Error iterating over consumer: %s", err)
		}

		sqlm, err := messages.NewSQL(qcm)
		if err != nil {
			log.Errorf("Error creating SQL Message: %s", err)
		}

		res, err := db.ExecContext(appCtx, sqlm.Content.Sentence)
		if err != nil {
			log.Errorf("Error storing SQL payload payload: %v", err)
		}

		log.Debugf("SQL message: %s", sqlm.ToJSON())
		log.Debugf("DB Execution Result: %s", res)

		// Notify main routine is done
		*done <- true
	}(&done)

	// main routine blocked until others routines finished
	<-done
}
