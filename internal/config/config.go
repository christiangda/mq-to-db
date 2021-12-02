package config

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// Consumer Default values
	DefaultConsumerAddress            = "localhost"
	DefaultConsumerPort               = 5672
	DefaultConsumerRequestedHeartbeat = time.Second * 10
	DefaultConsumerQueueExclusive     = false
	DefaultConsumerQueueAutoACK       = false

	// Database Default values
	DefaultDatabaseAddress         = "localhost"
	DefaultDatabasePort            = 5432
	DefaultDatabaseSSLMode         = "disable"
	DefaultDatabaseDatabase        = "postgres"
	DefatulDatabaseUsername        = "postgres"
	DefaultDatabasePassword        = "postgres"
	DefaultDatabaseMaxPingTimeOut  = time.Second * 1
	DefaultDatabaseMaxQueryTimeOut = time.Second * 10
	DefaultDatabaseConnMaxLifetime = time.Second * 0
	DefaultDatabaseMaxIdleConns    = 10
	DefaultDatabaseMaxOpenConns    = 20

	DefaultDispatcherConsumerConcurrency = 1
	DefaultDispatcherStorageWorkers      = 5
)

func New() Config {
	return Config{
		Consumer: Consumer{
			Address:            DefaultConsumerAddress,
			Port:               DefaultConsumerPort,
			RequestedHeartbeat: DefaultConsumerRequestedHeartbeat,
			Queue: ConsumerQueue{
				Exclusive: DefaultConsumerQueueExclusive,
				AutoACK:   DefaultConsumerQueueAutoACK,
			},
		},
		Database: Database{
			Address:         DefaultDatabaseAddress,
			Port:            DefaultDatabasePort,
			SSLMode:         DefaultDatabaseSSLMode,
			Database:        DefaultDatabaseDatabase,
			Username:        DefatulDatabaseUsername,
			Password:        DefaultDatabasePassword,
			MaxPingTimeOut:  DefaultDatabaseMaxPingTimeOut,
			MaxQueryTimeOut: DefaultDatabaseMaxQueryTimeOut,
			ConnMaxLifetime: DefaultDatabaseConnMaxLifetime,
			MaxIdleConns:    DefaultDatabaseMaxIdleConns,
			MaxOpenConns:    DefaultDatabaseMaxOpenConns,
		},
		Dispatcher: Dispatcher{
			ConsumerConcurrency: DefaultDispatcherConsumerConcurrency,
			StorageWorkers:      DefaultDispatcherStorageWorkers,
		},
	}
}

// This is a key value storage type used into some properties of the conf
type args map[string]interface{}

// Config is the structure with all configuration
type Config struct {
	Server      Server      `mapstructure:"server" json:"server" yaml:"server"`
	Dispatcher  Dispatcher  `json:"dispatcher" yaml:"dispatcher"`
	Consumer    Consumer    `json:"consumer" yaml:"consumer"`
	Database    Database    `json:"database" yaml:"database"`
	Application Application `json:"application" yaml:"application"`
}

// ToJSON export the configuration in JSON format
func (c *Config) ToJSON() string {
	out, err := json.Marshal(c)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}

// ToYAML export the configuration in YAML format
func (c *Config) ToYAML() string {
	out, err := yaml.Marshal(c)
	if err != nil {
		log.Panic(err)
	}
	return string(out)
}

type Server struct {
	Address           string        `json:"address" yaml:"address"`
	Port              int           `json:"port" yaml:"port"`
	ReadTimeout       time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout      time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
	IdleTimeout       time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`
	ShutdownTimeout   time.Duration `json:"shutdownTimeout" yaml:"shutdownTimeout"`
	KeepAlivesEnabled bool          `json:"keepAlivesEnabled" yaml:"keepAlivesEnabled"`
	LogFormat         string        `json:"logFormat" yaml:"logFormat"`
	LogLevel          string        `json:"logLevel" yaml:"logLevel"`
	Debug             bool          `json:"debug" yaml:"debug"`
	Profile           bool          `json:"profile" yaml:"profile"`
}

type Dispatcher struct {
	ConsumerConcurrency int `json:"consumerConcurrency" yaml:"consumerConcurrency"`
	StorageWorkers      int `json:"storageWorkers" yaml:"storageWorkers"`
}

type Consumer struct {
	Address            string           `json:"address" yaml:"address"`
	Port               int              `json:"port" yaml:"port"`
	RequestedHeartbeat time.Duration    `json:"requestedHeartbeat" yaml:"requestedHeartbeat"`
	Username           string           `json:"username" yaml:"username"`
	Password           string           `json:"password" yaml:"password"`
	VirtualHost        string           `json:"virtualHost" yaml:"virtualHost"`
	Queue              ConsumerQueue    `json:"queue" yaml:"queue"`
	Exchange           ConsumerExchange `json:"exchange" yaml:"exchange"`
}

type ConsumerQueue struct {
	Name          string `json:"name" yaml:"name"`
	RoutingKey    string `json:"routingKey" yaml:"routingKey"`
	Durable       bool   `json:"durable" yaml:"durable"`
	AutoDelete    bool   `json:"autoDelete" yaml:"autoDelete"`
	Exclusive     bool   `json:"exclusive" yaml:"exclusive"`
	AutoACK       bool   `json:"autoACK" yaml:"autoACK"`
	PrefetchCount int    `json:"prefetchCount" yaml:"prefetchCount"`
	PrefetchSize  int    `json:"prefetchSize" yaml:"prefetchSize"`
	Args          args   `json:"args" yaml:"args"`
}

type ConsumerExchange struct {
	Name       string `json:"name" yaml:"name"`
	Kind       string `mapstructure:"type" json:"type" yaml:"type"` // mapstructure is needed because the field into the config file is type and we are changing to kind
	Durable    bool   `json:"durable" yaml:"durable"`
	AutoDelete bool   `json:"autoDelete" yaml:"autoDelete"`
	Args       args   `json:"args" yaml:"args"`
}

type Database struct {
	Address  string `json:"address" yaml:"address"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
	SSLMode  string `json:"sslMode" yaml:"sslMode"`

	MaxPingTimeOut  time.Duration `json:"maxPingTimeOut" yaml:"maxPingTimeOut"`
	MaxQueryTimeOut time.Duration `json:"maxQueryTimeOut" yaml:"maxQueryTimeOut"`
	ConnMaxLifetime time.Duration `json:"connMaxLifetime" yaml:"connMaxLifetime"`
	MaxIdleConns    int           `json:"maxIdleConns" yaml:"maxIdleConns"`
	MaxOpenConns    int           `json:"maxOpenConns" yaml:"maxOpenConns"`
}

type Application struct {
	Name             string `json:"name" yaml:"name"`
	Description      string `json:"description" yaml:"description"`
	GitRepository    string `json:"gitRepository" yaml:"gitRepository"`
	Version          string `json:"version" yaml:"version"`
	Revision         string `json:"revision" yaml:"revision"`
	Branch           string `json:"branch" yaml:"branch"`
	BuildUser        string `json:"buildUser" yaml:"buildUser"`
	BuildDate        string `json:"buildDate" yaml:"buildDate"`
	GoVersion        string `json:"goVersion" yaml:"goVersion"`
	VersionInfo      string `json:"versionInfo" yaml:"versionInfo"`
	BuildInfo        string `json:"buildInfo" yaml:"buildInfo"`
	ConfigFile       string `json:"configFile" yaml:"configFile"`
	HealthPath       string `json:"healthPath" yaml:"healthPath"`
	MetricsPath      string `json:"metricsPath" yaml:"metricsPath"`
	MetricsNamespace string `json:"metricsNamespace" yaml:"metricsNamespace"`
}
