package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/consumer/rmq"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/repository"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/christiangda/mq-to-db/internal/storage/pgsql"

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

func init() {
	appHost, _ = os.Hostname()

	conf = config.New()

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
	flag.StringVar(&conf.Server.Address, "server.address", "", "Server address, empty means all address")
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
	flag.StringVar(&conf.Server.LogLevel, "logLevel", "info", "Log Level [panic|fatal|error|warn|info|debug|trace]")

	// Application conf flags
	flag.StringVar(&conf.Application.ConfigFile, "configFile", "config", "Configuration file")

	// Application version flags
	showVersion := flag.Bool("version", false, "Show application version")
	showVersionInfo := flag.Bool("versionInfo", false, "Show application version information")
	showBuildInfo := flag.Bool("buildInfo", false, "Show application build information")

	flag.Parse()
	// necessary to read from Env Vars too
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

	switch strings.ToLower(conf.Server.LogFormat) {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	case "text":
		log.SetFormatter(&log.TextFormatter{})
	default:
		log.Warnf("unknown log format: %s, using text", conf.Server.LogFormat)
		log.SetFormatter(&log.TextFormatter{DisableColors: false, DisableTimestamp: false, FullTimestamp: true})
	}

	if conf.Server.Debug {
		conf.Server.LogLevel = "debug"
	}

	// set the configured log level
	if level, err := log.ParseLevel(strings.ToLower(conf.Server.LogLevel)); err == nil {
		log.SetLevel(level)
	} else {
		log.Errorf("invalid log level %s", err)
	}

	log.Info("Application initialized")
}

func main() {
	log.Info("Starting application")

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
	// v.AllowEmptyEnv(true)
	// Substitute the _ to .
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

	// Select the storage
	log.Info("Using postgresql database")
	db, err := pgsql.New(&storage.Config{
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

	log.Info("Using rabbitmq consumer")
	qc, err := rmq.New(&consumer.Config{
		Name:               conf.Application.Name,
		Address:            conf.Consumer.Address,
		Port:               conf.Consumer.Port,
		RequestedHeartbeat: conf.Consumer.RequestedHeartbeat,
		Username:           conf.Consumer.Username,
		Password:           conf.Consumer.Password,
		VirtualHost:        conf.Consumer.VirtualHost,
		Queue: consumer.Queue{
			Name:          conf.Consumer.Queue.Name,
			RoutingKey:    conf.Consumer.Queue.RoutingKey,
			Durable:       conf.Consumer.Queue.Durable,
			AutoDelete:    conf.Consumer.Queue.AutoDelete,
			Exclusive:     conf.Consumer.Queue.Exclusive,
			AutoACK:       conf.Consumer.Queue.AutoACK,
			PrefetchCount: conf.Consumer.Queue.PrefetchCount,
			PrefetchSize:  conf.Consumer.Queue.PrefetchSize,
			Args:          conf.Consumer.Queue.Args,
		},
		Exchange: consumer.Exchange{
			Name:       conf.Consumer.Exchange.Name,
			Kind:       conf.Consumer.Exchange.Kind,
			Durable:    conf.Consumer.Exchange.Durable,
			AutoDelete: conf.Consumer.Exchange.AutoDelete,
			Args:       conf.Consumer.Exchange.Args,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Try to connects to Storage first, and if everything is ready, then go for Consumer
	log.Infof("Connecting to database")
	if err := db.Connect(appCtx); err != nil {
		log.WithFields(log.Fields{
			"server": conf.Database.Address,
			"port":   conf.Database.Port,
		}).Fatal("Error connecting to database server")
	}

	// Try to connect to queue consumer
	log.Infof("Connecting to queue")
	if err := qc.Connect(); err != nil {
		log.WithFields(log.Fields{
			"server":   conf.Consumer.Address,
			"port":     conf.Consumer.Port,
			"queue":    conf.Consumer.Queue.Name,
			"exchange": conf.Consumer.Exchange.Name,
		}).Fatal("Error connecting to queue server")
	}

	// repository to called it every time we need to store a consumer.Message into the database
	msgsRepo := repository.NewMessageRepository(appCtx, db, mtrs)

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
	sliceChanStorageWorkers := make([]<-chan repository.Results, conf.Dispatcher.StorageWorkers)

	// Start Consumers
	log.WithFields(log.Fields{"concurrency": conf.Dispatcher.ConsumerConcurrency}).Infof("Starting consumers")
	for i := 0; i < conf.Dispatcher.ConsumerConcurrency; i++ {
		// ids for consumers
		id := fmt.Sprintf("%s-%s-consumer-%d", appHost, conf.Application.Name, i)
		sliceChanMessages[i] = messageConsumer(appCtx, id, qc)
	}

	// Merge all channels from consumers in only one channel of type <-chan consumer.Messages
	chanMessages := mergeMessagesChans(appCtx, sliceChanMessages...)

	// Start storage workers
	log.WithFields(log.Fields{"concurrency": conf.Dispatcher.StorageWorkers}).Infof("Starting storage workers")
	for i := 0; i < conf.Dispatcher.StorageWorkers; i++ {
		// ids for storage workers
		id := fmt.Sprintf("%s-%s-storage-worker-%d", appHost, conf.Application.Name, i)
		sliceChanStorageWorkers[i] = messageProcessor(appCtx, id, chanMessages, msgsRepo)
	}

	// Merge all channels from workers in only one channel of type <-chan repository.Results
	chanResults := mergeResultsChans(appCtx, sliceChanStorageWorkers...)

	// Listen result in different routine
	go func() {
		for r := range chanResults {
			if r.Error != nil {
				log.WithFields(log.Fields{
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
	mux.HandleFunc("/", HomePageHandler)
	// health check handler
	mux.HandleFunc(conf.Application.HealthPath, HealthCheckHandler)

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
		Addr:              conf.Server.Address + ":" + strconv.Itoa(conf.Server.Port),
		Handler:           mux,
	}
	httpServer.SetKeepAlivesEnabled(conf.Server.KeepAlivesEnabled)

	// start httpserver in a go routine
	go func() {
		log.WithFields(log.Fields{
			"server": conf.Server.Address,
			"port":   conf.Server.Port,
		}).Info("Starting http server")

		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.WithFields(log.Fields{
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
	// TODO: define the buffer size
	out := make(chan consumer.Messages, 10)
	go func() {
		defer close(out)

		log.WithFields(log.Fields{"consumer": id}).Infof("Starting consumer")

		// prometheus metrics
		mtrs.ConsumerRunning.With(prometheus.Labels{"name": id}).Inc()

		// reading from message queue
		msgs, err := qc.Consume(id)
		if err != nil {
			log.Error(err)
		}

		// loop to dispatch the messages read to the channel consummed from storage workers
	loop:
		for {
			select {
			case msg, ok := <-msgs: // put messages consumed into the out chan
				if !ok {
					log.Warn("messageConsumer: Channel closed")
					break loop
				}

				mtrs.ConsumerMessages.With(prometheus.Labels{"name": id}).Inc()
				out <- msg

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(log.Fields{"consumer": id}).Warnf("Stoping consumer")
				mtrs.ConsumerRunning.With(prometheus.Labels{"name": id}).Dec()
				qc.Close() // TODO: Close the consumer connection not the mq connection

				break loop // go out of the for loop
			}
		}
	}()

	return out
}

// This function consume messages from queue system and return the messages as a channel of them
func messageProcessor(
	ctx context.Context,
	id string,
	chanMsgs <-chan consumer.Messages,
	st *repository.MessageRepository,
) <-chan repository.Results {
	// TODO: define the buffer size
	out := make(chan repository.Results, 10)
	go func() {
		defer close(out)

		log.WithFields(log.Fields{"worker": id}).Infof("Starting storage worker")

		// prometheus metrics
		mtrs.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Inc()

		// loop to dispatch the messages read to the channel consummed from storage workers
	loop:
		for {
			select {
			case m, ok := <-chanMsgs:
				if !ok {
					log.Warn("messageProcessor: Channel closed")
					break loop
				}

				startTime := time.Now()

				r := st.Store(m) // process and storage message into db
				r.By = id        // fill who execute it
				out <- r

				mtrs.StorageWorkerMessages.With(prometheus.Labels{"name": id}).Inc()

				mtrs.StorageWorkerProcessingDuration.With(
					prometheus.Labels{"name": id},
				).Observe(time.Since(startTime).Seconds())

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(log.Fields{"worker": id}).Warnf("Stoping storage worker")
				mtrs.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Dec()

				break loop // go out of the for loop
			}
		}
	}()

	return out
}

// this function merge all the channels data receive as slice of channels and return a merged channel with the data
// bassically convert (...<-chan consumer.Messages) --> (<-chan consumer.Messages)
func mergeMessagesChans(ctx context.Context, channels ...<-chan consumer.Messages) <-chan consumer.Messages {
	var wg sync.WaitGroup

	// TODO: define the buffer size
	out := make(chan consumer.Messages, 10)

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
func mergeResultsChans(ctx context.Context, channels ...<-chan repository.Results) <-chan repository.Results {
	var wg sync.WaitGroup

	// TODO: define the buffer size
	out := make(chan repository.Results, 10)

	// internal function to merge channels in only one
	multiplex := func(channel <-chan repository.Results) {
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
func HomePageHandler(w http.ResponseWriter, r *http.Request) {
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

	<h3>
        <a href="https://prometheus.io/">
            If you want to know more about Metrics and Exporters go to https://prometheus.io
        </a>
    </h3>
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
		log.Errorf("error rendering template: %s", err)
	}
}

// HealthCheck render the health check endpoint for the whole application
// TODO: Implement the health check, when database fail or consumer fail
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}
