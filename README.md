# mq-to-db

Read from Message Queue System and Store into Database

## How to execute

source code

```bash
git clone https://github.com/christiangda/mq-to-db.git
cd mq-to-db/
go run -race  ./cmd/mq-to-db/main.go --help
```

__NOTE:__ the parameter `-race`is to check [race conditions](https://blog.golang.org/race-detector) because we are using [Go Concurrency](https://blog.golang.org/pipelines)

binary

```bash
./mq-to-db --help
```

## Internal References

* [Configuration](docs/config.md)
* [Messages Type](docs/messages.md)

## External References

### Free books

* https://golang.org/doc/effective_go.html#generality

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

*  [https://www.rabbitmq.com/tutorials/tutorial-one-go.html](https://www.rabbitmq.com/tutorials/tutorial-one-go.html)
*  [http://www.inanzzz.com/index.php/post/0aeg/creating-a-rabbitmq-producer-example-with-golang](http://www.inanzzz.com/index.php/post/0aeg/creating-a-rabbitmq-producer-example-with-golang)