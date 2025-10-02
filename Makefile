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
	rm -f almalinux/context/install-go

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
	cp bin/$(BINARY) almalinux/context/install-go

.PHONY: test
test:
	GO_ENV=testing go test -race -cover ./...

.PHONY: debian
debian: binary
	SEMVER_MAJOR_MINOR=12 docker buildx bake --no-cache --set=*.output=type=docker debian
	OS_NAME=debian CIS_VERSION=v4.1-4 SEMVER_MAJOR_MINOR=12 ./test.sh ghcr.io/metal-stack/debian:12

.PHONY: nvidia
nvidia:
	SEMVER_MAJOR_MINOR=12 docker buildx bake --no-cache --set=*.output=type=docker debian-nvidia

.PHONY: ubuntu
ubuntu: binary
	SEMVER_MAJOR_MINOR=24.04 docker buildx bake --no-cache --set=*.output=type=docker ubuntu
	OS_NAME=ubuntu SEMVER_MAJOR_MINOR=24.04 ./test.sh ghcr.io/metal-stack/ubuntu:24.04

.PHONY: firewall
firewall: binary
	SEMVER_MAJOR_MINOR=3.0-ubuntu docker buildx bake --no-cache --set=*.output=type=docker ubuntu-firewall
	OS_NAME=firewall SEMVER_MAJOR_MINOR=3.0-ubuntu ./test.sh ghcr.io/metal-stack/firewall:3.0-ubuntu

.PHONY: almalinux
almalinux: binary
	SEMVER_MAJOR_MINOR=9 docker buildx bake --no-cache --set=*.output=type=docker almalinux
	OS_NAME=almalinux SEMVER_MAJOR_MINOR=9 ./test.sh ghcr.io/metal-stack/almalinux:9
