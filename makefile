BASE := mindl
OUTDIR := bin
OUT = ${OUTDIR}/${BASE}${EXT}
PKG := github.com/MinoMino/mindl
VERSION := $(shell git describe --always --long --dirty --tags)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/)

# Make sure the executable has .exe on Windows.
ifeq ($(OS),Windows_NT)
	EXT := .exe
else
	EXT :=
endif

BUILDFLAGS := -ldflags="-w -s -X main.version=${VERSION}-${BRANCH}"
DEBUGBUILDFLAGS := -ldflags="-X main.version=${VERSION}-${BRANCH}"

all: build

build:
	@mkdir -p ${OUTDIR}
	go build -i -o ${OUT} ${BUILDFLAGS} ${PKG}
	@echo "Done!"
	@echo "The executable can be found in: ${OUT}"

build-debug:
	@mkdir -p ${OUTDIR}
	go build -i -o ${OUT} ${DEBUGBUILDFLAGS} ${PKG}
	@echo "Done!"
	@echo "The executable can be found in: ${OUT}"

install:
	go install ${BUILDFLAGS} ${PKG}
	@echo "Done!"
	@echo "Add GOPATH/bin to your PATH environment variable if you haven't already and run it with: ${BASE}${EXT}"

test:
	@go test -short ${PKG_LIST}

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ;  do \
		golint $$file ; \
	done

run: build
	./${OUT}

clean:
	-@rm ${OUT}

# Cross-compilation stuff.
EXTRAFILES := THIRD-PARTY-NOTICES README.md
DISTDIR := dist
BUILDCALL = env GOOS=${1} GOARCH=${2} go build -i -o ${OUTDIR}/${BASE}${3} ${BUILDFLAGS} ${PKG}
ARCHIVECALL = python archive.py ${1} ${DISTDIR}/${BASE}-${2} ${OUTDIR}/${BASE}${3} ${EXTRAFILES}
DISTCLEANCALL = rm ${OUTDIR}/${BASE}${1}

dist: build-osx32 build-osx64 build-linux32 build-linux64 build-windows32 build-windows64

distdir:
	@mkdir -p ${DISTDIR}

build-osx32: distdir
	@echo "Compiling for OSX 32-bit..."
	@$(call BUILDCALL,darwin,386,)
	@$(call ARCHIVECALL,tar,osx32,)
	@$(call DISTCLEANCALL,)

build-osx64: distdir
	@echo "Compiling for OSX 64-bit..."
	@$(call BUILDCALL,darwin,amd64,)
	@$(call ARCHIVECALL,tar,osx64,)
	@$(call DISTCLEANCALL,)

build-linux32: distdir
	@echo "Compiling for Linux 32-bit..."
	@$(call BUILDCALL,linux,386,)
	@$(call ARCHIVECALL,tar,linux32,)
	@$(call DISTCLEANCALL,)

build-linux64: distdir
	@echo "Compiling for Linux 64-bit..."
	@$(call BUILDCALL,linux,amd64,)
	@$(call ARCHIVECALL,tar,linux64,)
	@$(call DISTCLEANCALL,)

build-windows32: distdir
	@echo "Compiling for Windows 32-bit..."
	@$(call BUILDCALL,windows,386,.exe)
	@$(call ARCHIVECALL,zip,windows32,.exe)
	@$(call DISTCLEANCALL,.exe)

build-windows64: distdir
	@echo "Compiling for Windows 64-bit..."
	@$(call BUILDCALL,windows,amd64,.exe)
	@$(call ARCHIVECALL,zip,windows64,.exe)
	@$(call DISTCLEANCALL,.exe)

.PHONY: run build build-debug static vet lint dist build-osx32 build-osx64 build-linux32 build-linux64 build-windows32 build-windows64
