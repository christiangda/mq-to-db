package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/consumer/kafka"
	"github.com/christiangda/mq-to-db/internal/consumer/rmq"
	"github.com/christiangda/mq-to-db/internal/logger"
	log "github.com/christiangda/mq-to-db/internal/logger"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/christiangda/mq-to-db/internal/storage/memory"
	"github.com/christiangda/mq-to-db/internal/storage/pgsql"
	"github.com/christiangda/mq-to-db/internal/storer"

	"os"
	"strings"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/version"
	"github.com/spf13/viper"
)

const (
	appName             = "mq-to-db"
	appDescription      = "This is a Message Queue Consumer to store the payload into Database"
	appGitRepository    = "https://github.com/christiangda/mq-to-db"
	appMetricsPath      = "/metrics"
	appMetricsNamespace = "mq_to_db"
	appHealthPath       = "/health"
)

var (
	appHost string
	conf    config.Config
	v       = viper.New()
	mtrs    *metrics.Metrics
)

func init() { // package initializer
	appHost, _ = os.Hostname()
	log.AddHook(logger.NewGlobalFieldsHook(appName, appHost, conf.Application.Version))

	// Set default values
	conf.Application.Name = appName
	conf.Application.Description = appDescription
	conf.Application.GitRepository = appGitRepository
	conf.Application.MetricsPath = appMetricsPath
	conf.Application.MetricsNamespace = appMetricsNamespace
	conf.Application.HealthPath = appHealthPath
	conf.Application.Version = version.Version
	conf.Application.Revision = version.Revision
	conf.Application.GoVersion = version.GoVersion
	conf.Application.BuildUser = version.BuildUser
	conf.Application.BuildDate = version.BuildDate
	conf.Application.VersionInfo = version.GetVersionInfo()
	conf.Application.BuildInfo = version.GetVersionInfoExtended()

	// Server conf flags
	flag.StringVar(&conf.Server.Address, "server.address", "", "Server address, empty means all address") //empty means all the address
	flag.IntVar(&conf.Server.Port, "server.port", 8080, "Server port")
	flag.DurationVar(&conf.Server.ReadTimeout, "server.readTimeout", 2*time.Second, "Server ReadTimeout")
	flag.DurationVar(&conf.Server.WriteTimeout, "server.writeTimeout", 5*time.Second, "Server WriteTimeout")
	flag.DurationVar(&conf.Server.IdleTimeout, "server.idleTimeout", 60*time.Second, "Server IdleTimeout")
	flag.DurationVar(&conf.Server.ReadTimeout, "server.readHeaderTimeout", 5*time.Second, "Server ReadHeaderTimeout")
	flag.DurationVar(&conf.Server.ShutdownTimeout, "server.shutdownTimeout", 30*time.Second, "Server ShutdownTimeout")
	flag.BoolVar(&conf.Server.KeepAlivesEnabled, "server.keepAlivesEnabled", true, "Server KeepAlivesEnabled")
	flag.BoolVar(&conf.Server.Debug, "debug", false, "debug")
	flag.BoolVar(&conf.Server.Profile, "profile", false, "Enable program profile")
	flag.StringVar(&conf.Server.LogFormat, "logFormat", "text", "Log Format [text|json]")
	flag.StringVar(&conf.Server.LogLevel, "logLevel", "info", "Log Level [debug|info|warning|panic|fatal]")

	// Application conf flags
	flag.StringVar(&conf.Application.ConfigFile, "configFile", "config", "Configuration file")

	// Application version flags
	showVersion := flag.Bool("version", false, "Show application version")
	showVersionInfo := flag.Bool("versionInfo", false, "Show application version information")
	showBuildInfo := flag.Bool("buildInfo", false, "Show application build information")

	flag.Parse()
	//necessary to read from Env Vars too
	if err := v.BindPFlags(flag.CommandLine); err != nil {
		log.Fatal(err)
	}

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

	// Logs conf
	if strings.ToLower(conf.Server.LogFormat) == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{DisableColors: false, DisableTimestamp: false, FullTimestamp: true})
	}

	switch conf.Server.LogLevel {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warning":
		log.SetLevel(logrus.WarnLevel)
	case "panic":
		log.SetLevel(logrus.PanicLevel)
	case "fatal":
		log.SetLevel(logrus.FatalLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	// if --debug, force debug log level no matter what value is in logLevel
	if conf.Server.Debug {
		log.SetLevel(logrus.DebugLevel)
	}

	log.Info("Application initialized")
}

func main() {

	log.Info("Starting application")

	// Viper default values to conf parameters when config file doesn't have it
	// The config file values overrides these

	// ***** Dispatcher *****
	// dispatcher.consumerConcurrency: 1
	// dispatcher.storageWorkers: 5
	v.SetDefault("dispatcher.consumerConcurrency", 1)
	v.SetDefault("dispatcher.storageWorkers", 5)

	// ***** DATABASE *****
	// database.kind: postgresql
	// database.port: 5432
	// database.sslMode: disable
	// database.maxPingTimeOut: 1s
	// database.maxQueryTimeOut: 10s
	// database.connMaxLifetime: 0
	// database.maxIdleConns: 5
	// database.maxOpenConns: 20
	v.SetDefault("database.kind", "postgresql")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslMode", "disable")
	v.SetDefault("Database.maxPingTimeOut", "1s")
	v.SetDefault("Database.maxQueryTimeOut", "10s")
	v.SetDefault("Database.connMaxLifetime", 0)
	v.SetDefault("Database.maxIdleConns", 5)
	v.SetDefault("Database.maxIdleConns", 20)
	// ***** RabbitMQ *****
	// consumer.workers: 10
	// consumer.kind: rabbitmq
	// consumer.port: 5672
	// consumer.requestedHeartbeat: 25
	// consumer.queue.autoACK: false
	v.SetDefault("consumer.workers", 10)
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
	log.Debugf("Environment Variables: %s", os.Environ())
	v.AutomaticEnv()
	//v.AllowEmptyEnv(true)
	//Substitute the _ to .
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // because EnvVar are SERVER_PORT and config is server.port

	log.Infof("Loading configuration file: %s", conf.Application.ConfigFile)
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	log.Info("Configuring application")
	if err := v.Unmarshal(&conf); err != nil {
		log.Fatalf("Unable to decode: %v", err)
	}

	log.Debugf("Application configuration: %s", conf.ToJSON())

	osSignal := make(chan bool, 1) // this channels will be used to listen OS signals, like ^c
	ListenOSSignals(&osSignal)     // this function as soon as receive an Operating System signals, put value in chan done

	appCtx := context.Background()
	appCtx, cancel := context.WithCancel(appCtx)
	mtrs = metrics.New(&conf)

	// Create abstraction layers (Using interfaces)
	var db storage.Store
	var qc consumer.Consumer

	var err error // Necessary to handle errors inside switch/case
	// Select the storage
	switch conf.Database.Kind {
	case "memory":
		db, err = memory.New(&storage.Config{})
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using  memory database")
	case "postgresql":
		db, err = pgsql.New(&storage.Config{
			Address:  conf.Database.Address,
			Port:     conf.Database.Port,
			Username: conf.Database.Username,
			Password: conf.Database.Password,
			Database: conf.Database.Database,
			SSLMode:  conf.Database.SSLMode,

			MaxPingTimeOut:  conf.Database.MaxPingTimeOut,
			MaxQueryTimeOut: conf.Database.MaxQueryTimeOut,
			ConnMaxLifetime: conf.Database.ConnMaxLifetime,
			MaxIdleConns:    conf.Database.MaxIdleConns,
			MaxOpenConns:    conf.Database.MaxOpenConns,
		}, mtrs)
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
		qc, err = kafka.New(&consumer.Config{})
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using kafka consumer")
	case "rabbitmq":
		qc, err = rmq.New(&consumer.Config{
			Name:               conf.Application.Name,
			Address:            conf.Consumer.Address,
			Port:               conf.Consumer.Port,
			RequestedHeartbeat: conf.Consumer.RequestedHeartbeat,
			Username:           conf.Consumer.Username,
			Password:           conf.Consumer.Password,
			VirtualHost:        conf.Consumer.VirtualHost,
			Queue: struct {
				Name          string
				RoutingKey    string
				Durable       bool
				AutoDelete    bool
				Exclusive     bool
				AutoACK       bool
				PrefetchCount int
				PrefetchSize  int
				Args          map[string]interface{}
			}{
				conf.Consumer.Queue.Name,
				conf.Consumer.Queue.RoutingKey,
				conf.Consumer.Queue.Durable,
				conf.Consumer.Queue.AutoDelete,
				conf.Consumer.Queue.Exclusive,
				conf.Consumer.Queue.AutoACK,
				conf.Consumer.Queue.PrefetchCount,
				conf.Consumer.Queue.PrefetchSize,
				conf.Consumer.Queue.Args,
			},
			Exchange: struct {
				Name       string
				Kind       string
				Durable    bool
				AutoDelete bool
				Args       map[string]interface{}
			}{
				conf.Consumer.Exchange.Name,
				conf.Consumer.Exchange.Kind,
				conf.Consumer.Exchange.Durable,
				conf.Consumer.Exchange.AutoDelete,
				conf.Consumer.Exchange.Args,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Info("Using rabbitmq consumer")
	default:
		log.Fatal("Inside configuration file consumer.kind must be [rabbitmq|kafka]")
	}

	// Try to connects to Storage first, and if everithing is ready, then go for Consumer
	log.Infof("Connecting to database")
	if err := db.Connect(appCtx); err != nil {
		log.WithFields(logrus.Fields{
			"server": conf.Database.Address,
			"port":   conf.Database.Port,
		}).Fatal("Error connecting to database server")
	}

	// Try to connect to queue consumer
	log.Infof("Connecting to queue")
	if err := qc.Connect(); err != nil {
		log.WithFields(logrus.Fields{
			"server":   conf.Consumer.Address,
			"port":     conf.Consumer.Port,
			"queue":    conf.Consumer.Queue.Name,
			"exchange": conf.Consumer.Exchange.Name,
		}).Fatal("Error connecting to queue server")
	}

	// storer to called it every time we need to store a consumer.Message into the database
	strer := storer.New(appCtx, db, mtrs)

	// Logic of channels for consumer and for storage
	// it is a go pipeline model https://blog.golang.org/pipelines
	// ********************************************

	// slice of consumers
	// where the consumers will put the chan of consumer.Messages
	// this slice of channels (consumer.Messages) is used to comunicate consumers and storage workers
	// but first every consumer generate a <-chan consumer.Messages witch need to me merge in only
	// one channel before be ready to consume
	sliceChanMessages := make([]<-chan consumer.Messages, conf.Dispatcher.ConsumerConcurrency)

	// Slice of workers
	sliceChanStorageWorkers := make([]<-chan storer.Results, conf.Dispatcher.StorageWorkers)

	// Start Consumers
	log.WithFields(logrus.Fields{"concurrency": conf.Dispatcher.ConsumerConcurrency}).Infof("Starting consumers")
	for i := 0; i < conf.Dispatcher.ConsumerConcurrency; i++ {
		// ids for consumers
		id := fmt.Sprintf("%s-%s-consumer-%d", appHost, conf.Application.Name, i)
		sliceChanMessages[i] = messageConsumer(appCtx, id, qc)
	}

	// Merge all channels from consumers in only one channel of type <-chan consumer.Messages
	chanMessages := mergeMessagesChans(appCtx, sliceChanMessages...)

	// Start storage workers
	log.WithFields(logrus.Fields{"concurrency": conf.Dispatcher.StorageWorkers}).Infof("Starting storage workers")
	for i := 0; i < conf.Dispatcher.StorageWorkers; i++ {
		// ids for storage workers
		id := fmt.Sprintf("%s-%s-storage-worker-%d", appHost, conf.Application.Name, i)
		sliceChanStorageWorkers[i] = messageProcessor(appCtx, id, chanMessages, strer)
	}

	// Merge all channels from workers in only one channel of type <-chan storer.Results
	chanResults := mergeResultsChans(appCtx, sliceChanStorageWorkers...)

	// Listen result in different routine
	go func() {
		for r := range chanResults {
			if r.Error != nil {
				log.WithFields(logrus.Fields{
					"worker": r.By,
				}).Errorf("%s-%s", r.Reason, r.Error)
			}
		}
	}()
	// ********************************************

	// Filling global metrics
	mtrs.Up.Add(1)
	mtrs.Info.Add(1)
	mtrs.EnableDBStats(db) // this enable the DB Stats collector for database/sql package

	// Expose metrics, health checks and home
	mux := http.NewServeMux()
	// metrics handler
	mux.Handle(conf.Application.MetricsPath, mtrs.GetHandler())
	// Home handler
	mux.HandleFunc("/", HomePage)
	// health check handler
	mux.HandleFunc(conf.Application.HealthPath, HealthCheck)

	// Profilling endpoints whe -profile or --profile
	if conf.Server.Profile {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/heap", pprof.Index)
		mux.HandleFunc("/debug/pprof/mutex", pprof.Index)
		mux.HandleFunc("/debug/pprof/goroutine", pprof.Index)
		mux.HandleFunc("/debug/pprof/threadcreate", pprof.Index)
		mux.HandleFunc("/debug/pprof/block", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	// http server conf
	httpServer := &http.Server{
		ReadTimeout:       conf.Server.ReadTimeout,
		WriteTimeout:      conf.Server.WriteTimeout,
		IdleTimeout:       conf.Server.IdleTimeout,
		ReadHeaderTimeout: conf.Server.ReadHeaderTimeout,
		Addr:              conf.Server.Address + ":" + strconv.Itoa(int(conf.Server.Port)),
		Handler:           mux,
	}
	httpServer.SetKeepAlivesEnabled(conf.Server.KeepAlivesEnabled)

	// start httpserver in a go routine
	go func() {

		log.WithFields(logrus.Fields{
			"server": conf.Server.Address,
			"port":   conf.Server.Port,
		}).Info("Starting http server")

		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.WithFields(logrus.Fields{
				"server": conf.Server.Address,
				"port":   conf.Server.Port,
			}).Errorf("Error starting http server %s", err)

		}
		osSignal <- true // make a gratefull shutdown
	}()

	// Block the main function here until we receive OS signals
	<-osSignal

	// call context cancellation
	log.Warn("Executing context cancellation, gracefully shutdown")
	cancel()

	// Closes sockets
	log.Warn("Closing Consumer connections")
	if err := qc.Close(); err != nil {
		log.Warnf("Consumer connections was closed previously: %s", err)
	}
	log.Warn("Consumer connections closed")

	log.Warn("Closing Database connections")
	if err := db.Close(); err != nil {
		log.Error(err)
	}
	log.Warn("Database connections closed")

	log.Warn("Application stopped")
}

// ListenOSSignals is a functions that
// start a go routine to listen Operating System Signals
// When a signal is received, it put a value inside channel done
// to notify main routine to close
func ListenOSSignals(osSignal *chan bool) {
	go func(osSignal *chan bool) {
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt)
		signal.Notify(osSignals, syscall.SIGTERM)
		signal.Notify(osSignals, syscall.SIGINT)
		signal.Notify(osSignals, syscall.SIGQUIT)

		log.Info("Listening Operating System signals")
		sig := <-osSignals // This go routine is blocked here until receive a OS Signal
		log.Warnf("Received signal %s from Operating System", sig)

		// Notify main routine that shutdown was solicited
		*osSignal <- true
	}(osSignal)
}

// This function consume messages from queue system and return the messages as a channel of them
func messageConsumer(ctx context.Context, id string, qc consumer.Consumer) <-chan consumer.Messages {
	out := make(chan consumer.Messages)
	go func() {
		defer close(out)

		log.WithFields(logrus.Fields{"consumer": id}).Infof("Starting consumer")

		//prometheus metrics
		mtrs.ConsumerRunning.With(prometheus.Labels{"name": id}).Inc()

		// reading from message queue
		msgs, err := qc.Consume(id)
		if err != nil {
			log.Error(err)
		}

		// loop to dispatch the messages read to the channel consummed from storage workers
		for m := range msgs {
			select {

			case out <- m: // put messages consumed into the out chan
				mtrs.ConsumerMessages.With(prometheus.Labels{"name": id}).Inc()

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(logrus.Fields{"consumer": id}).Warnf("Stoping consumer")
				mtrs.ConsumerRunning.With(prometheus.Labels{"name": id}).Dec()

				// closes the consumer queue and connection
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					qc.Close() // TODO: Close the consumer connection not the mq connection
					wg.Done()
				}()
				wg.Wait()

				return // go out of the for loop
			}
		}
	}()

	return out
}

// This function consume messages from queue system and return the messages as a channel of them
func messageProcessor(ctx context.Context, id string, chanMsgs <-chan consumer.Messages, st storer.Storer) <-chan storer.Results {
	out := make(chan storer.Results)
	go func() {
		defer close(out)

		log.WithFields(logrus.Fields{"worker": id}).Infof("Starting storage worker")

		//prometheus metrics
		mtrs.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Inc()

		// loop to dispatch the messages read to the channel consummed from storage workers
		for {
			select {

			case m := <-chanMsgs:
				startTime := time.Now()

				r := st.Store(m) // proccess and storage message into db
				r.By = id        // fill who execute it
				out <- r

				mtrs.StorageWorkerMessages.With(prometheus.Labels{"name": id}).Inc()

				mtrs.StorageWorkerProcessingDuration.With(
					prometheus.Labels{"name": id},
				).Observe(time.Since(startTime).Seconds())

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(logrus.Fields{"worker": id}).Warnf("Stoping storage worker")
				mtrs.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Dec()

				return // go out of the for loop

			}
		}
	}()

	return out
}

// this function merge all the channels data receive as slice of channels and return a merged channel with the data
// bassically convert (...<-chan consumer.Messages) --> (<-chan consumer.Messages)
func mergeMessagesChans(ctx context.Context, channels ...<-chan consumer.Messages) <-chan consumer.Messages {
	var wg sync.WaitGroup
	out := make(chan consumer.Messages)

	// internal function to merge channels in only one
	multiplex := func(channel <-chan consumer.Messages) {
		defer wg.Done()
		for ch := range channel {
			select {
			case out <- ch:
			case <-ctx.Done():
				return
			}
		}
	}

	// one routine per every channel in channels arg
	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	// waiting until fished every go multiplex(ch)
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// mergeResultsChan merge all the channels of storer.Results in only one
// bassically convert (...<-chan storer.Results) --> (<-chan storer.Results)
func mergeResultsChans(ctx context.Context, channels ...<-chan storer.Results) <-chan storer.Results {
	var wg sync.WaitGroup
	out := make(chan storer.Results)

	// internal function to merge channels in only one
	multiplex := func(channel <-chan storer.Results) {
		defer wg.Done()
		for ch := range channel {
			select {
			case out <- ch:
			case <-ctx.Done():
				return
			}
		}
	}

	// one routine per every channel in channels arg
	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	// waiting until fished every go multiplex(ch)
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// HomePage render the home page website
func HomePage(w http.ResponseWriter, r *http.Request) {
	indexHTMLTmpl := `
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1><a href="{{.GitRepository}}">{{.Name}}</a></h1>
	<h2>{{.Description}}</h2>
	<h3>Links:</h3>
	<ul>
		<li><a href="{{.MetricsPath}}">{{.MetricsPath}}</a></li>
		<li><a href="{{.HealthPath}}">{{.HealthPath}}</a></li>
	</ul>

	<h2>Version</h2>
	<ul>
		<li>{{.VersionInfo}}</li>
		<li>{{.BuildInfo}}</li>
	</ul>

	<h2>Go profile is enabled</h2>
	<ul>
		<li><a href="{{.ProfileLink}}">{{.ProfileLink}}</a></li>
	</ul>


	<h3><a href="https://prometheus.io/">If you want to know more about Metrics and Exporters go to https://prometheus.io</a></h3>
</body>
</html>
`
	data := struct {
		Title         string
		Name          string
		Description   string
		GitRepository string
		MetricsPath   string
		HealthPath    string
		VersionInfo   string
		BuildInfo     string
		ProfileLink   string
	}{
		conf.Application.Name,
		conf.Application.Name,
		conf.Application.Description,
		conf.Application.GitRepository,
		conf.Application.MetricsPath,
		conf.Application.HealthPath,
		conf.Application.VersionInfo,
		conf.Application.BuildInfo,
		"/debug/pprof/",
	}

	t := template.Must(template.New("index").Parse(indexHTMLTmpl))
	if err := t.Execute(w, data); err != nil {
		log.Errorf("Error rendering template: %s", err)
	}
}

// HealthCheck render the health check endpoint for the whole application
// TODO: Implement the health check, when database fail or consumer fail
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}
