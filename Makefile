# assertoor
VERSION := $(shell git rev-parse --short HEAD)
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo.BuildVersion="$(VERSION)"'
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo.BuildRelease="$(RELEASE)"'

.PHONY: all test clean

all: build

test:
	go test ./...

build:
	@echo version: $(VERSION)
	go build -v -o bin/ -ldflags="-s -w $(GOLDFLAGS)" .

clean:
	rm -f bin/*
