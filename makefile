ifeq ($(OS),Windows_NT)
	OUT := bin/mindl.exe
else
	OUT := bin/mindl
endif
PKG := github.com/MinoMino/mindl
VERSION := $(shell git describe --always --long --dirty --tags)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

all: build

build:
	go build -i -v -o ${OUT} -ldflags="-X main.version=${VERSION}-${BRANCH}" ${PKG}

test:
	@go test -short ${PKG_LIST}

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

static: vet lint
	go build -i -v -o ${OUT}-v${VERSION}-${BRANCH} -ldflags="-extldflags \"-static\" -w -s -X main.version=${VERSION}-${BRANCH}" ${PKG}

run: build
	./${OUT}

clean:
	-@rm ${OUT} ${OUT}-v*

.PHONY: run build static vet lint