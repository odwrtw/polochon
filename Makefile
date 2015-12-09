PACKAGE := github.com/xunien/polochon
SUBPACKAGES := \
	$(PACKAGE)/modules/eztv \
	$(PACKAGE)/modules/addicted \
	$(PACKAGE)/modules/openguessit \
	$(PACKAGE)/modules/fsnotify \
	$(PACKAGE)/modules/canape \
	$(PACKAGE)/modules/inotify \
	$(PACKAGE)/modules/yts \
	$(PACKAGE)/modules/tvdb \
	$(PACKAGE)/modules/transmission \
	$(PACKAGE)/modules/opensubtitles \
	$(PACKAGE)/modules/pushover \
	$(PACKAGE)/modules/kickass \
	$(PACKAGE)/modules/imdb \
	$(PACKAGE)/modules/yifysubs \
	$(PACKAGE)/modules/tmdb \
	$(PACKAGE)/lib

COVERPROFILES := \
	imdb.coverprofile \
	opensubtitles.coverprofile \
	yifysubs.coverprofile \
	tvdb.coverprofile \
	eztv.coverprofile \
	tmdb.coverprofile \
	yts.coverprofile \
	lib.coverprofile

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

gover.coverprofile: $(COVERPROFILES)
	gover

goveralls: gover.coverprofile
	$(HOME)/gopath/bin/goveralls -coverprofile=gover.coverprofile -service=travis-ci

imdb.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/imdb

opensubtitles.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/opensubtitles

yifysubs.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/yifysubs

tvdb.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/tvdb

eztv.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/eztv

tmdb.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/tmdb

yts.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/modules/yts

lib.coverprofile:
	$(GO) test -v -coverprofile=$@ $(GOBUILD_LDFLAGS) $(PACKAGE)/lib

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
	$(RM) -v coverage.html $(COVERPROFILES)
	$(GO) clean $(SUBPACKAGES) || true
	if [ -d $${GOPATH%%:*}/pkg ] ; then \
		find $${GOPATH%%:*}/pkg -wholename '*xunien/polochon*.a' | xargs $(RM) -fv || true; \
	fi

.PHONY: fmt
fmt:
	set -e; for f in $(shell git ls-files '*.go'); do gofmt $$f | diff -u $$f - ; done
