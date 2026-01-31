# assertoor
VERSION := $(shell git rev-parse --short HEAD)
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/buildinfo.BuildVersion="$(VERSION)"'
GOLDFLAGS += -X 'github.com/ethpandaops/assertoor/pkg/buildinfo.BuildRelease="$(RELEASE)"'
CURRENT_UID := $(shell id -u)
CURRENT_GID := $(shell id -g)

.PHONY: all docs test clean

all: docs build

test:
	go test ./...

build:
	@echo version: $(VERSION)
	env CGO_ENABLED=1 go build -v -o bin/ -ldflags="-s -w $(GOLDFLAGS)" .

docs:
	go install github.com/swaggo/swag/cmd/swag@v1.16.3 && swag init -g web/api/handler.go -d pkg --parseDependency -o pkg/web/api/docs

clean:
	rm -f bin/*

devnet:
	.hack/devnet/run.sh

devnet-run: devnet
	go run main.go --config .hack/devnet/generated-assertoor-config.yaml --verbose

devnet-run-docker: devnet
	docker build --file ./Dockerfile-local -t assertoor:devnet-run --build-arg userid=$(CURRENT_UID) --build-arg groupid=$(CURRENT_GID) .
	docker run --rm -v $(PWD):/workspace -p 8080:8080 -u $(CURRENT_UID):$(CURRENT_GID) --network kt-assertoor -it assertoor:devnet-run --config .hack/devnet/generated-assertoor-config.yaml

devnet-clean:
	.hack/devnet/cleanup.sh
