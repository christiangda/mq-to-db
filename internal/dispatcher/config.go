package dispatcher

// DispatcherConfig is the configuration for the Dispatcher.
type Config struct {
	ApplicationName     string `json:"applicationName" yaml:"applicationName"`
	HostName            string `json:"hostname" yaml:"hostname"`
	ConsumerConcurrency int    `json:"consumerConcurrency" yaml:"consumerConcurrency"`
	StorageWorkers      int    `json:"storageWorkers" yaml:"storageWorkers"`
}
