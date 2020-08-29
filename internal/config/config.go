package config

import (
	"encoding/json"
	"time"

	log "github.com/christiangda/mq-to-db/internal/logger"
	"gopkg.in/yaml.v3"
)

// This is a key value storage type used into some properties of the conf
type args map[string]interface{}

// Config is the structure with all configuration
//
// how is the name into
// the config file
//     |                  |           |
// `mapstructure:"type" json:"type" yaml:"type"`
type Config struct {

	// This part is private, will be filled using flags, the file values overrides flags values
	Server struct {
		Address           string        `json:"address" yaml:"address"`
		Port              int           `json:"port" yaml:"port"`
		ReadTimeout       time.Duration `json:"readTimeout" yaml:"readTimeout"`
		WriteTimeout      time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
		IdleTimeout       time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
		ReadHeaderTimeout time.Duration `json:"readHeaderTimeout" yaml:"readHeaderTimeout"`
		ShutdownTimeout   time.Duration `json:"shutdownTimeout" yaml:"shutdownTimeout"`
		KeepAlivesEnabled bool          `json:"keepAlivesEnabled" yaml:"keepAlivesEnabled"`
		LogFormat         string        `json:"logFormat" yaml:"logFormat"`
		Debug             bool          `json:"debug" yaml:"debug"`
	}

	Dispatcher struct {
		ConsumerConcurrency int `json:"consumerConcurrency" yaml:"consumerConcurrency"`
		StorageWorkers      int `json:"storageWorkers" yaml:"storageWorkers"`
	}

	Consumer struct {
		Kind               string        `json:"kind" yaml:"kind"`
		Address            string        `json:"address" yaml:"address"`
		Port               int           `json:"port" yaml:"port"`
		RequestedHeartbeat time.Duration `json:"requestedHeartbeat" yaml:"requestedHeartbeat"`
		Username           string        `json:"username" yaml:"username"`
		Password           string        `json:"password" yaml:"password"`
		VirtualHost        string        `json:"virtualHost" yaml:"virtualHost"`
		Queue              struct {
			Name       string `json:"name" yaml:"name"`
			RoutingKey string `json:"routingKey" yaml:"routingKey"`
			Durable    bool   `json:"durable" yaml:"durable"`
			AutoDelete bool   `json:"autoDelete" yaml:"autoDelete"`
			Exclusive  bool   `json:"exclusive" yaml:"exclusive"`
			AutoACK    bool   `json:"autoACK" yaml:"autoACK"`
			Args       args   `json:"args" yaml:"args"`
		} `json:"Queue" yaml:"queue"`
		Exchange struct {
			Name       string `json:"name" yaml:"name"`
			Kind       string `mapstructure:"type" json:"type" yaml:"type"` // mapstructure is needed because the field into the config file is type and we are changing to kind
			Durable    bool   `json:"durable" yaml:"durable"`
			AutoDelete bool   `json:"autoDelete" yaml:"autoDelete"`
			Args       args   `json:"args" yaml:"args"`
		} `json:"exchange" yaml:"exchange"`
	}

	Database struct {
		Kind     string `json:"kind" yaml:"kind"`
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

	// This part is private, will be filled using code, not from file
	Application struct {
		Name          string `json:"name" yaml:"name"`
		Description   string `json:"description" yaml:"description"`
		GitRepository string `json:"gitRepository" yaml:"gitRepository"`
		Version       string `json:"version" yaml:"version"`
		Revision      string `json:"revision" yaml:"revision"`
		Branch        string `json:"branch" yaml:"branch"`
		BuildUser     string `json:"buildUser" yaml:"buildUser"`
		BuildDate     string `json:"buildDate" yaml:"buildDate"`
		GoVersion     string `json:"goVersion" yaml:"goVersion"`
		VersionInfo   string `json:"versionInfo" yaml:"versionInfo"`
		BuildInfo     string `json:"buildInfo" yaml:"buildInfo"`
		ConfigFile    string `json:"configFile" yaml:"configFile"`
		HealthPath    string `json:"healthPath" yaml:"healthPath"`
		MetricsPath   string `json:"metricsPath" yaml:"metricsPath"`
	}
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
