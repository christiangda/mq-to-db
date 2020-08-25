# mq-to-db

Read from Message Queue System and Store into Database

## Characteristics

* The process (job) of consume one message from queue and store into the database is synchronous because every message needs to be acknowledge (confirm as storage).

## How to execute

source code

```bash
git clone https://github.com/christiangda/mq-to-db.git
cd mq-to-db/
go run -race  ./cmd/mq-to-db/main.go --help

# and then
go run -race  ./cmd/mq-to-db/main.go --configFile config-sample.yaml
```

__NOTE:__ the parameter `-race`is to check [race conditions](https://blog.golang.org/race-detector) because we are using [Go Concurrency](https://blog.golang.org/pipelines)

binary

```bash
./mq-to-db --help
```

RabbitMQ

```bash
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

__NOTE:__  

* RabbitMQ web console: [http://localhost:15672](http://localhost:15672)
* Username: guest
* Password: guest

PostgreSQL

```bash
docker run --rm  --name postgresql -e POSTGRES_PASSWORD=mysecretpassword -p 5432:5432 -d postgres

# logs
docker logs postgresql -f

# remember to  stop and remove (--rm in docker run do it for you)
docker stop postgresql
```

mq-to-db

```bash
go run -race  ./cmd/mq-to-db/main.go --configFile config-sample.yaml
```

o

```bash
make
./mq-to-db --configFile config-sample.yaml
```

## How to build

```bash
go build \
    -o mq-to-db \
    -ldflags "-X github.com/christiangda/mq-to-db/internal/version.Version=$(git rev-parse --abbrev-ref HEAD) -X github.com/christiangda/mq-to-db/internal/version.Revision=$(git rev-parse HEAD) -X github.com/christiangda/mq-to-db/internal/version.Branch=$(git rev-parse --abbrev-ref HEAD) -X github.com/christiangda/mq-to-db/internal/version.BuildUser=\"$(git config --get user.name | tr -d '\040\011\012\015\n')\" -X github.com/christiangda/mq-to-db/internal/version.BuildDate=$(date +'%Y-%m-%dT%H:%M:%S')" \
    ./cmd/mq-to-db/main.go
```

## Internal References

* [Configuration](docs/config.md)
* [Messages Type](docs/messages.md)

## External References

### Free books

* [https://www.openmymind.net/The-Little-Go-Book/](https://www.openmymind.net/The-Little-Go-Book/)
* https://golang.org/doc/effective_go.html#generality
* [https://golang.org/doc/code.html](https://golang.org/doc/code.html)

### Blogs

* [https://www.goin5minutes.com/screencasts/](https://www.goin5minutes.com/screencasts/)

### Databases

* [https://golang.org/pkg/database/sql/](https://golang.org/pkg/database/sql/)
* [https://golang.org/s/sqldrivers](https://golang.org/s/sqldrivers)
* [https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/05.4.html](https://astaxie.gitbooks.io/build-web-application-with-golang/content/en/05.4.html)
* [https://gist.github.com/divan/eb11ddc97aab765fb9b093864410fd25](https://gist.github.com/divan/eb11ddc97aab765fb9b093864410fd25)

### Project Layout

* [https://github.com/golang-standards/project-layout](https://github.com/golang-standards/project-layout)

### Interfaces

* https://www.alexedwards.net/blog/interfaces-explained
* https://golang.org/doc/effective_go.html#interfaces_and_types

### Context

* [https://www.sohamkamani.com/golang/2018-06-17-golang-using-context-cancellation/](https://www.sohamkamani.com/golang/2018-06-17-golang-using-context-cancellation/)

### YAML|JSON to Struct

* [https://github.com/go-yaml/yaml](https://github.com/go-yaml/yaml)
* [https://www.sohamkamani.com/golang/2018-07-19-golang-omitempty/](https://www.sohamkamani.com/golang/2018-07-19-golang-omitempty/)
* [https://ubuntu.com/blog/api-v3-of-the-yaml-package-for-go-is-available](https://ubuntu.com/blog/api-v3-of-the-yaml-package-for-go-is-available)

### Config Files, Flags, Env Vars

* [https://blog.gopheracademy.com/advent-2014/configuration-with-fangs/](https://blog.gopheracademy.com/advent-2014/configuration-with-fangs/)

### RabbitMQ

* [https://www.rabbitmq.com/queues.html#optional-arguments](https://www.rabbitmq.com/queues.html#optional-arguments)
* [https://www.rabbitmq.com/dlx.html](https://www.rabbitmq.com/dlx.html)
* [https://www.rabbitmq.com/vhosts.html](https://www.rabbitmq.com/vhosts.html)
* [https://www.rabbitmq.com/tutorials/tutorial-one-go.html](https://www.rabbitmq.com/tutorials/tutorial-one-go.html)
* [http://www.inanzzz.com/index.php/post/0aeg/creating-a-rabbitmq-producer-example-with-golang](http://www.inanzzz.com/index.php/post/0aeg/creating-a-rabbitmq-producer-example-with-golang)

### Logs

* [https://github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)

### Metrics

* [https://prometheus.io/docs/guides/go-application/](https://prometheus.io/docs/guides/go-application/)

### Test

* [https://www.sidorenko.io/post/2019/01/testing-of-functions-with-channels-in-go/](https://www.sidorenko.io/post/2019/01/testing-of-functions-with-channels-in-go/)

### Variable Injection

* [https://blog.alexellis.io/inject-build-time-vars-golang/](https://blog.alexellis.io/inject-build-time-vars-golang/)
* [https://goenning.net/2017/01/25/adding-custom-data-go-binaries-compile-time/](https://goenning.net/2017/01/25/adding-custom-data-go-binaries-compile-time/)

### Iterators

* [https://ewencp.org/blog/golang-iterators/](https://ewencp.org/blog/golang-iterators/)
* [https://pkg.go.dev/google.golang.org/api/iterator?tab=doc#example-package-ServerPages](https://pkg.go.dev/google.golang.org/api/iterator?tab=doc#example-package-ServerPages)
* [https://github.com/googleapis/google-cloud-go/wiki/Iterator-Guidelines](https://github.com/googleapis/google-cloud-go/wiki/Iterator-Guidelines)

### Concurrency

* [https://www.goin5minutes.com/blog/channel_over_channel/](https://www.goin5minutes.com/blog/channel_over_channel/)

### Workers Pool

* [https://brandur.org/go-worker-pool](https://brandur.org/go-worker-pool)
* [https://maptiks.com/blog/using-go-routines-and-channels-with-aws-2/](https://maptiks.com/blog/using-go-routines-and-channels-with-aws-2/)