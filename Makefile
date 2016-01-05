PACKAGE := github.com/odwrtw/polochon

VERSION_VAR  := main.VersionString
REPO_VERSION := $(shell git describe --always --dirty --tags)

REV_VAR  := main.RevisionString
REPO_REV := $(shell git rev-parse -q HEAD)

GO  ?= go
GOX ?= gox
GOBUILD_LDFLAGS := -ldflags "\
	-X '$(VERSION_VAR)=$(REPO_VERSION)' \
	-X '$(REV_VAR)=$(REPO_REV)' \
"
GOBUILD_FLAGS ?=
GOTEST_FLAGS  ?=
GOX_OSARCH    ?= linux/amd64 linux/arm darwin/amd64
GOX_FLAGS     ?= -output="builds/polochon_{{.OS}}_{{.Arch}}" -osarch="$(GOX_OSARCH)" -parallel=3

TRAVIS_BUILD_DIR ?= .
export TRAVIS_BUILD_DIR

.PHONY: all
all: clean test

.PHONY: test
test: build fmt .test

.PHONY: quicktest
quicktest:
	$(GO) test $(GOTEST_FLAGS) $(SUBPACKAGES)

.PHONY: .test
.test: coverage.html

coverage.html: gover.coverprofile
	$(GO) tool cover -html=$^ -o $@

gover.coverprofile:
	set -e; for pkg in $(shell find -name "*.go" -printf "%h\n" | sort -u); do $(GO) test -v $$pkg -coverprofile $$pkg.coverprofile ; done && gover

goveralls: gover.coverprofile
	$(HOME)/gopath/bin/goveralls -coverprofile=gover.coverprofile -service=travis-ci

.PHONY: build
build: deps .build

.PHONY: .build
.build:
	$(GO) build $(GOBUILD_FLAGS) $(GOBUILD_LDFLAGS) ./...

.PHONY: crossbuild
crossbuild: deps
	$(GOX) $(GOX_FLAGS) $(GOBUILD_FLAGS) $(GOBUILD_LDFLAGS) ./...

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
