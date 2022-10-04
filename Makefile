.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)
BINARY := install

LINKMODE := -extldflags=-static \
		 -X 'github.com/metal-stack/v.Version=$(VERSION)' \
		 -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
		 -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
		 -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'

.PHONY: binary
binary: test
	GGO_ENABLED=0 \
		go build \
			-trimpath \
			-tags osusergo,netgo \
			-o bin/$(BINARY) \
			-ldflags "$(LINKMODE)" \
		github.com/metal-stack/metal-images/cmd
	strip bin/$(BINARY)
	cp bin/$(BINARY) debian/context/install-go
	cp bin/$(BINARY) centos/context/install-go

.PHONY: test
test:
	GO_ENV=testing go test -cover ./...

