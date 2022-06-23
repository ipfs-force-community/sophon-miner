SHELL=/usr/bin/env bash

all: build
.PHONY: all docker

GOVERSION:=$(shell go version | cut -d' ' -f 3 | cut -d. -f 2)
ifeq ($(shell expr $(GOVERSION) \< 17), 1)
$(warning Your Golang version is go 1.$(GOVERSION))
$(error Update Golang to version to at least 1.17.9)
endif

# git modules that need to be loaded
MODULES:=

CLEAN:=
BINS:=

ldflags=-X=github.com/filecoin-project/venus-miner/build.CurrentCommit='+git$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))'
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

## FFI

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build/.filecoin-install ;

build/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build/.filecoin-install
CLEAN+=build/.filecoin-install

$(MODULES): build/.update-modules ;

# dummy file that marks the last time modules were updated
build/.update-modules:
	git submodule update --init --recursive
	touch $@

# end git modules

## MAIN BINARIES

CLEAN+=build/.update-modules

deps: $(BUILD_DEPS)
.PHONY: deps

build: miner
.PHONY: build

deps: $(BUILD_DEPS)
.PHONY: deps

miner: $(BUILD_DEPS)
	rm -f venus-miner
	go build $(GOFLAGS) -o venus-miner ./cmd/

.PHONY: miner
BINS+=venus-miner

docsgen:
	go build $(GOFLAGS) -o docgen-md ./api/docgen
	./docgen-md > ./docs/en/api-v0-methods-miner.md
	rm docgen-md

# MISC

buildall: $(BINS)

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
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


docker:
	docker build --build-arg https_proxy=$(BUILD_DOCKER_PROXY) -t venus-miner .
