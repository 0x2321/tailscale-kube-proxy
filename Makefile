MODULE := tailscale-kube-proxy
VERSION := $(shell git describe --tags)
COMMIT := $(shell git rev-parse --short HEAD)
TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

build:
	CGO_ENABLED=0 go build \
        -ldflags "-X '$(MODULE)/cmd.version=$(VERSION)' \
                  -X '$(MODULE)/cmd.commit=$(COMMIT)' \
                  -X '$(MODULE)/cmd.time=$(TIME)'"