# assertoor
VERSION := $(shell git rev-parse --short HEAD)
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo.BuildVersion="$(VERSION)"'
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo.BuildRelease="$(RELEASE)"'

.PHONY: all docs test clean

all: docs build

test:
	go test ./...

build:
	@echo version: $(VERSION)
	env CGO_ENABLED=1 go build -v -o bin/ -ldflags="-s -w $(GOLDFLAGS)" .

docs:
	go install github.com/swaggo/swag/cmd/swag@v1.16.3 && swag init -g web/api/handler.go -d pkg/coordinator --parseDependency -o pkg/coordinator/web/api/docs

clean:
	rm -f bin/*

devnet:
	.hack/devnet/run.sh

devnet-run: devnet
	go run main.go --config .hack/devnet/generated-assertoor-config.yaml

devnet-run-docker: devnet build
	docker build --file ./Dockerfile-stub -t assertoor:devnet-run .
	docker run --rm -v $(PWD):/app --entrypoint "./bin/assertoor" -it assertoor:devnet-run --config .hack/devnet/generated-assertoor-config.yaml

devnet-clean:
	.hack/devnet/cleanup.sh
