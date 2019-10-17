PACKAGE := github.com/odwrtw/polochon

VERSION_VAR  := main.VersionString
REPO_VERSION := $(shell git describe --always --dirty --tags)

REV_VAR  := main.RevisionString
REPO_REV := $(shell git rev-parse -q HEAD)

GO       ?= go
GOX      ?= gox
BIN_NAME ?= polochon
GOBUILD_LDFLAGS := -ldflags "\
	-X '$(VERSION_VAR)=$(REPO_VERSION)' \
	-X '$(REV_VAR)=$(REPO_REV)' \
"
GOX_DEFAULT_OS        ?= $(shell $(GO) env GOOS)
GOX_DEFAULT_ARCH      ?= $(shell $(GO) env GOARCH)
GOX_CROSS_OSARCH_FLAG ?= -osarch="linux/amd64 linux/arm darwin/amd64"
GOX_OSARCH_FLAG       ?= -osarch="$(GOX_DEFAULT_OS)/$(GOX_DEFAULT_ARCH)"
GOX_OUTPUT_FLAG       ?= -output="builds/$(BIN_NAME)_{{.OS}}_{{.Arch}}"
GOX_PARALLEL_FLAG     ?= -parallel=3

TRAVIS_BUILD_DIR ?= .
export TRAVIS_BUILD_DIR

.PHONY: all
all: clean test

.PHONY: test
test: build fmt .test

.PHONY: .test
.test: coverage.html

coverage.html: gover.coverprofile
	$(GO) tool cover -html=$^ -o $@

gover.coverprofile:
	set -e; \
	$(GO) list -f '"echo {{ .ImportPath }} && $(GO) test -v -coverprofile={{ .Dir }}/{{.Name}}.coverprofile {{ .ImportPath }}"' ./... | xargs -L1 sh -c && \
	gover

goveralls: gover.coverprofile
	$(HOME)/gopath/bin/goveralls -coverprofile=gover.coverprofile -service=travis-ci

.PHONY: build
build: deps .build

.PHONY: .build
.build:
	$(GOX) $(GOX_OUTPUT_FLAG) $(GOX_OSARCH_FLAG) $(GOBUILD_LDFLAGS) ./...

.PHONY: crossbuild
crossbuild: deps
	$(GOX) $(GOX_OUTPUT_FLAG) $(GOX_CROSS_OSARCH_FLAG) $(GOX_PARALLEL_FLAG) $(GOBUILD_LDFLAGS) ./...

.PHONY: deps
deps: .gox-install .goveralls-install .gover-install

.gox-install:
	$(GO) get -x github.com/mitchellh/gox

.goveralls-install:
	$(GO) get golang.org/x/tools/cmd/cover
	$(GO) get github.com/mattn/goveralls
	$(GO) get github.com/axw/gocov/gocov

.gover-install:
	$(GO) get golang.org/x/tools/cmd/cover
	$(GO) get github.com/modocache/gover

.PHONY: distclean
distclean: clean
	$(RM) -rv ./builds

.PHONY: clean
clean:
	$(RM) -v coverage.html
	find . -name "*.coverprofile" -exec rm {} \;
	$(GO) clean -x ./... || true

.PHONY: fmt
fmt:
	set -e; for f in $(shell git ls-files '*.go'); do gofmt $$f | diff -u $$f - ; done

.PHONY: checksum
checksum:
	set -e; cd builds; for f in polochon_*; do sha256sum $$f > "$$f.sha256sum"; done
