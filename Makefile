.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)
BINARY := install


.PHONY: all
bin/$(BINARY):
	GGO_ENABLED=0 \
	GO111MODULE=on \
		go build \
			-trimpath \
			-tags netgo \
			-o bin/$(BINARY) \
			-ldflags "-X 'github.com/metal-stack/v.Version=$(VERSION)' \
					  -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
					  -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
					  -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" cmd/install.go && strip bin/$(BINARY)

.PHONY: test
test:
	GO_ENV=testing go test -cover ./...

