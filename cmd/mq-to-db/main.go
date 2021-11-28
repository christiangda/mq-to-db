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
	"syscall"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/christiangda/mq-to-db/internal/dispatcher"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/queue"
	"github.com/christiangda/mq-to-db/internal/repository"
	"github.com/christiangda/mq-to-db/internal/storage"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/version"
	_ "github.com/lib/pq" // this is the way to load pgsql driver to be used by golang database/sql
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

	// set default values
	conf = config.New()

	// Set Application values
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
	flag.StringVar(&conf.Server.Address, "server.address", "", "Server address, empty means all address") // empty means all the address
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
	if level, err := log.ParseLevel(conf.Server.LogLevel); err == nil {
		log.SetLevel(level)
	} else {
		log.Errorf("invalid log level %s", err)
	}

	log.Info("Application initialized")
}

func main() {
	log.Info("Starting application")

	// Viper default values to conf parameters when config file doesn't have it
	// The config file values overrides these

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
	db, err := storage.NewPGSQL(&storage.Config{
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
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Using rabbitmq consumer")
	qc, err := queue.NewRabbitMQ(&queue.Config{
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

	// Try to connects to Storage first, and if everithing is ready, then go for Consumer
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

	// repository used to store a consumer.Message into the database
	msgsRepo := repository.NewMessageRepository(appCtx, db, mtrs)

	// pseudo code
	distpatcherConfig := dispatcher.Config{
		ApplicationName:     conf.Application.Name,
		HostName:            conf.Application.Name,
		ConsumerConcurrency: conf.Dispatcher.ConsumerConcurrency,
		StorageWorkers:      conf.Dispatcher.StorageWorkers,
	}

	consumer := dispatcher.NewConsumer(context.TODO(), qc, distpatcherConfig)

	storer := dispatcher.NewStorer(context.TODO(), msgsRepo, distpatcherConfig)

	go func() {
		chanResults := storer.Store(consumer.Consume())
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
	}()
	// end of pseudo code

	// ********************************************

	// Filling global metrics
	mtrs.Up.Add(1)
	mtrs.Info.Add(1)

	// Expose metrics, health checks and home
	mux := http.NewServeMux()
	// metrics handler
	mux.Handle(conf.Application.MetricsPath, metricsHandler())
	// Home handler
	mux.HandleFunc("/", homePageHandler)
	// health check handler
	mux.HandleFunc(conf.Application.HealthPath, healthCheckHandler)

	// Profilling endpoints when use -profile or --profile
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

// HomePage render the home page website
func homePageHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Errorf("error rendering template: %s", err)
	}
}

// HealthCheck render the health check endpoint for the whole application
// TODO: Implement the health check, when database fail or consumer fail
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func metricsHandler() http.Handler {
	return promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}
