SHELL=/usr/bin/env bash

all: build
.PHONY: all docker

GOVERSION:=$(shell go version | cut -d' ' -f 3 | cut -d. -f 2)
ifeq ($(shell expr $(GOVERSION) \< 17), 1)
$(warning Your Golang version is go 1.$(GOVERSION))
$(error Update Golang to version to at least 1.18.1)
endif

CLEAN:=
BINS:=

BUILD_TARGET=sophon-miner

ldflags=-X=github.com/ipfs-force-community/sophon-miner/build.CurrentCommit='+git$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))'
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

build: miner
.PHONY: build

miner:
	rm -f $(BUILD_TARGET)
	go build $(GOFLAGS) -o $(BUILD_TARGET) ./cmd/

get-beacon:
	go build -o get-beacon ./cmd/get-beacon/
.PHONY: get-beacon

debug:
	rm -f $(BUILD_TARGET)
	go build $(GOFLAGS) -gcflags=all="-N -l" -o $(BUILD_TARGET) ./cmd/

.PHONY: miner
BINS+=sophon-miner

docsgen:
	go build $(GOFLAGS) -o docgen-md ./api/docgen
	./docgen-md > ./docs/en/api-v0-methods-miner.md
	rm docgen-md

# MISC

buildall: $(BINS)

clean:
	rm -rf $(CLEAN) $(BINS)
.PHONY: clean

dist-clean:
	git clean -xdff
	git submodule deinit --all -f
.PHONY: dist-clean

gen:
	go run ./gen/api
	goimports -w api
.PHONY: gen

print-%:
	@echo $*=$($*)


# docker
.PHONY: docker

mock:
	go run github.com/golang/mock/mockgen -destination=./miner/mock/mock_post_provider.go  -source=./miner/util.go -package mock

TAG:=test
docker:
ifdef DOCKERFILE
	cp $(DOCKERFILE) ./dockerfile
else
	curl -o dockerfile https://raw.githubusercontent.com/filecoin-project/venus-docs/master/script/docker/dockerfile
endif
	docker build --build-arg https_proxy=$(BUILD_DOCKER_PROXY) --build-arg BUILD_TARGET=$(BUILD_TARGET)  -t sophon-miner .
	docker tag sophon-miner filvenus/sophon-miner:$(TAG)
ifdef PRIVATE_REGISTRY
	docker tag sophon-miner $(PRIVATE_REGISTRY)/filvenus/sophon-miner:$(TAG)
endif

docker-push: docker
	docker push $(PRIVATE_REGISTRY)/filvenus/sophon-miner:$(TAG)
