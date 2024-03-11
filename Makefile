.ONESHELL:
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))
BINARY := install

LINKMODE := -extldflags=-static \
		 -X 'github.com/metal-stack/v.Version=$(VERSION)' \
		 -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
		 -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
		 -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'


all: clean binary

.PHONY: clean
clean:
	rm -f debian/context/install-go
	rm -f centos/context/install-go

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
	cp bin/$(BINARY) almalinux/context/install-go

.PHONY: test
test:
	GO_ENV=testing go test -race -cover ./...

.PHONY: debian
debian: binary
	docker-make -nNL -w debian -f docker-make.debian.yaml

.PHONY: ubuntu
ubuntu: binary
	docker-make -nNL -w debian -f docker-make.ubuntu.yaml

.PHONY: firewall
firewall: ubuntu
	docker-make -nNL -w firewall -f docker-make.yaml

.PHONY: centos
centos: binary
	docker-make -nNL -w centos -f docker-make.yaml
