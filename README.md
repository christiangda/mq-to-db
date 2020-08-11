# mq-to-db

Read from Message Queue System and Store into Database

## How to execute

source code

```bash
git clone https://github.com/christiangda/mq-to-db.git
cd mq-to-db/
go run -race  ./cmd/mq-to-db/main.go --help
```

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