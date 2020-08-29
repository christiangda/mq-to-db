# Configuration

This is about how to fill the configuration file for the service

The configuration file needed by the application need to be in `yaml` format.

The configuration file could be `named as you want`, by default if you don't specified the parameter `--configFile` is `config.yaml`
This could be located in any place, but by default when service start this find the configuration file in the following location:

1. `./config.yaml` Next to the service binary `mq-to-db`
2. `/etc/mq-to-db/config.yaml` Into the standard `Linux` configuration filesystem
3. `$HOME/config.yaml`  Into the `user home` which is executing the service binary `mq-to-db`

__NOTE:__ Remember that thanks to the parameter `--configFile` you can tell to the service which  configuration file you want to use, for example:

1. `mq-to-db --configFile '/tmp/myconfig.yaml'`
2. `mq-to-db --configFile './mq-to-db.yaml'`

To read the config file we are using the `golang` package [viper](https://github.com/spf13/viper) and [pflag](https://github.com/spf13/pflag)

## config.yaml

The config file [config-sample.yaml](/config-sample.yaml) could be used as template

```yaml
---
dispatcher:
  consumerConcurrency: 4 # Number of go routines consuming messages from Queue
  storageWorkers: 30  # Number of go routines processing the messages received from consumers and sending messages to storage

consumer:
  kind: rabbitmq
  address: 127.0.0.1
  port: 5672
  requestedHeartbeat: 25s
  username: guest
  password: guest
  virtualHost: my.virtualhost # Optional
  queue:
    name: my.queue
    routingKey: my.routeKey
    durable: true
    autoDelete: true
    exclusive: false
    autoACK: false
    args:                    # Optional
      x-message-ttl: 180000
      x-dead-letter-exchange: retry.exchange
  exchange:
    name: my.exchage
    type: topic
    durable: true
    autoDelete: false
    args:                    # Optional
      alternate-exchange: my.ae

database:
  kind: postgresql
  address: 127.0.0.1
  port: 5432
  username: postgres
  password: mysecretpassword
  database: postgres
  sslMode: disable
  maxPingTimeOut: 1s
  maxQueryTimeOut: 10s
  connMaxLifetime: 0
  maxIdleConns: 5
  maxOpenConns: 40
```
