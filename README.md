
# graylog-state

Idempotent Graylog API client enabling easy IaC configuration.

## Development notes

Use the included `docker-compose.yml` to create a development environment:

    $ docker-compose up

Enter the Go container with:

    $ docker exec -it graylogstate_state_1 bash

In the container, first fetch any dependencies and then run or build the program:

    $ go get github.com/davecgh/go-spew/spew
    $ go get github.com/dghubble/sling
    $ go get gopkg.in/yaml.v2
    ...
    $ CGO_ENABLED=0 go build -a -ldflags "-s -w" graylog-state.go
