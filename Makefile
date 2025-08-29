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
	cd debian; docker buildx bake --no-cache debian-12
	OS_NAME=${OS_NAME} CIS_VERSION=${CIS_VERSION} SEMVER=${SEMVER_MAJOR_MINOR}${SEMVER_PATCH} ../test.sh ghcr.io/metal-stack/${OS_NAME}:${SEMVER}
	OS_NAME=${OS_NAME} SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR} SEMVER_PATCH=${SEMVER_PATCH} ../export.sh

.PHONY: nvidia
nvidia:
	cd debian-nvidia; SEMVER_PATCH=${SEMVER_PATCH} SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR} docker buildx bake --no-cache debian-nvidia
	OS_NAME=${OS_NAME} SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR} SEMVER_PATCH=${SEMVER_PATCH} ../export.sh

.PHONY: ubuntu
ubuntu: binary
	cd debian; docker buildx bake --no-cache ubuntu-2404
	OS_NAME=${OS_NAME} SEMVER=${SEMVER_MAJOR_MINOR}${SEMVER_PATCH} ../test.sh ghcr.io/metal-stack/${OS_NAME}:${SEMVER}
	OS_NAME=${OS_NAME} SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR} SEMVER_PATCH=${SEMVER_PATCH} ../export.sh

.PHONY: firewall
firewall: ubuntu
	docker-make -nNL -w firewall -f docker-make.yaml

.PHONY: almalinux
almalinux: binary
	docker-make -nNL -w almalinux -f docker-make.yaml
