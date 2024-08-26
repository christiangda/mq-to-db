# mq-to-db (message queue to database)

![Release workflow](https://github.com/christiangda/mq-to-db/workflows/Release%20workflow/badge.svg)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/christiangda/mq-to-db?style=plastic)
[![Go Report Card](https://goreportcard.com/badge/github.com/christiangda/mq-to-db)](https://goreportcard.com/report/github.com/christiangda/mq-to-db)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/christiangda/mq-to-db?label=github%20release&style=plastic)
![Docker Pulls](https://img.shields.io/docker/pulls/christiangda/mq-to-db?label=docker%20hub%20pulls&style=plastic)
![Docker Image Version (latest semver)](https://img.shields.io/docker/v/christiangda/mq-to-db?label=docker%20hub%20tag&style=plastic)
[![CodeQL Analysis](https://github.com/christiangda/mq-to-db/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/christiangda/mq-to-db/actions/workflows/codeql-analysis.yml)
[![golangci-lint](https://github.com/christiangda/mq-to-db/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/christiangda/mq-to-db/actions/workflows/golangci-lint.yml)

This is a [Golang (go)](https://golang.org/) program to read [a specific JSON Payload message](docs/messages.md) from a Message Queue System and Store into Database using concurrency

This is a close image of how it works:

![mq-to-db](images/nxconsumers-mxworkers.jpg)

## Consumers supported

* [RabbitMQ](https://www.rabbitmq.com/)

## Storage supported

* [PostgreSQL](https://www.postgresql.org/)

## Characteristics

* The number of queue consumers could be different from the numbers of storage workers, see [config-sample.yaml](https://github.com/christiangda/mq-to-db/blob/main/config-sample.yaml)
* The process (job) of consuming one message from the queue and store into the database is synchronous because every message needs to be acknowledged (confirmed as stored).
* [Golang](https://golang.org/pkg/net/http/pprof/) `pprof` enabled via `--profile` command line when starting the service
* Prometheus metrics for consumers, storage workers, go statistics, and database
* [Grafana](https://grafana.com/) dashboard for [Prometheus.io](https://prometheus.io/) metrics
* [Dockerfile](Dockerfile) multi-stage build
* Makefile to facilitate the project builds
* [docker-compose file](https://github.com/christiangda/mq-to-db/blob/main/docker-compose.yaml) and [configuration](https://github.com/christiangda/mq-to-db/tree/main/docker-compose) to testing all elements
* docker images at [docker-hub](https://hub.docker.com/repository/docker/christiangda/mq-to-db) and [Github Packages](https://github.com/christiangda/mq-to-db/packages)
* [CI/CD Github Action pipeline](https://github.com/christiangda/mq-to-db/actions) workflow

## How to execute

There are many ways to do it, but always is necessary `PostgreSQL` and `RabbitMQ` dependencies, the easy way to see how it works is using containers.

### docker-compose

The program and all dependencies and visibility systems at once

#### Up

```bash
docker-compose up --build
```

#### Down

```bash
docker-compose down -v
```

#### Available links

After docker-compose start all the services, you have the following links ready to be curious, I prepared a simple `grafana dashboard` to show you part of the `prometheus metrics` implemented

* [mq-to-db-01 home page](http://localhost:8080/)
* [mq-to-db-02 home page](http://localhost:8081/)
* [Prometheus Dashboard](http://localhost:9090/)
* [Grafana Dashboard](http://localhost:3000/)
* [RabbitMQ Dashboard](http://localhost:15672/)

#### Profiling

```bash
# terminal 1, for mq-to-db-01 inside the docker-compose-file
go tool pprof http://127.0.0.1:8080/debug/pprof/goroutine

# terminal 2, for mq-to-db-02 inside the docker-compose-file
go tool pprof http://127.0.0.1:8081/debug/pprof/goroutine

# once you are into tool pprof, execute the command web
(pprof) web
```

#### See the Logs

```bash
docker-compose logs mq-to-db-01
docker-compose logs mq-to-db-02
```

### Manually

First install dependencies

#### RabbitMQ

```bash
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

__NOTES:__

* RabbitMQ web console: [http://localhost:15672](http://localhost:15672)
* Username: guest
* Password: guest

#### PostgreSQL

```bash
docker run --rm  --name postgresql -e POSTGRES_PASSWORD=mysecretpassword -p 5432:5432 -d postgres

# logs
docker logs postgresql -f

# remember to  stop and remove (--rm in docker run do it for you)
docker stop postgresql
```

### Using source code

```bash
git clone https://github.com/christiangda/mq-to-db.git
cd mq-to-db/
go run -race ./cmd/mq-to-db/main.go --help

# and then
go run -race ./cmd/mq-to-db/main.go --configFile config-sample.yaml
```

__NOTE:__ the parameter `-race`is to check [race conditions](https://blog.golang.org/race-detector) because we are using [Go Concurrency](https://blog.golang.org/pipelines)

### How to build

compiling for your ARCH and OS

```bash
# make executable first
make

# check the available options
./build/mq-to-db --help

# execute
./build/mq-to-db --configFile config-sample.yaml
```

Cross-compiling

```bash
make build-dist

# cross-compiling files
ls -l ./dist/
```

__NOTES__ related to make

* 1. This create a cross-compiling binaries and also Docker Image (linux 64bits)
* 2. Check the [Makefile](Makefile) to see `The make available targets options`
* 3. Remember to start dependencies first

### Using docker image

Here I use `latest tag`, but you can see all [releases here](https://hub.docker.com/repository/docker/christiangda/mq-to-db/tags?page=1)

```bash
# pull the image first
docker pull christiangda/mq-to-db:latest

# see available option
docker run --rm --name mq-to-db christiangda/mq-to-db:latest - --help

# run with a config file mapped and with profile option
docker run --rm -v <path to config file>:/etc/mq-to-db/config.yaml --name mq-to-db christiangda/mq-to-db:latest - --profile
```

__NOTES:__

* Remember to start dependencies first

### Available endpoints

The application expose different endpoints via http server

* [http://localhost:8080/](http://localhost:8080/)
* [http://localhost:8080/metrics](http://localhost:8080/metrics)
* [http://localhost:8080/health](http://localhost:8080/health)
* [http://localhost:8080/debug/pprof](http://localhost:8080/debug/pprof)

## References

Internals

* [Configuration](docs/config.md)
* [Messages Type](docs/messages.md)

Externals

* [references](docs/references.md)

## License

This module is released under the GNU General Public License Version 3:

* [http://www.gnu.org/licenses/gpl-3.0-standalone.html](http://www.gnu.org/licenses/gpl-3.0-standalone.html)

## Author Information

* [Christian González Di Antonio](https://github.com/christiangda)
