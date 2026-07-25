VERSION ?= 0.1.0-dev
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := \
	-X github.com/aliamerj/icl/cli.Version=$(VERSION) \
	-X github.com/aliamerj/icl/cli.Commit=$(COMMIT) \
	-X github.com/aliamerj/icl/cli.Date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o icl ./cmd/icl/main.go
