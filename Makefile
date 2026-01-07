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
	sudo umount test/rootfs/sys/firmware/efi/efivars
	sudo umount test/rootfs/sys
	sudo umount test/rootfs/proc
	sudo umount test/rootfs/dev
	sudo umount test/rootfs
	rm -rf test/rootfs
	rm -f test/disk.raw
	rm -f test/initramfs
	rm -f test/metal-kernel
	rm -f test/os-kernel
	rm -rf images
	rm -rf bin

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
	mkdir -p "images/debian/12"
	OS_NAME=debian OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=12 docker buildx bake --no-cache debian
	OS_NAME=debian OUTPUT_FOLDER="" CIS_VERSION=v4.1-4 SEMVER_MAJOR_MINOR=12 ./test.sh

.PHONY: nvidia
nvidia:
	mkdir -p "images/nvidia/12"
	OS_NAME=nvidia OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=12 docker buildx bake --no-cache debian-nvidia

.PHONY: ubuntu
ubuntu: binary
	mkdir -p "images/ubuntu/24.04"
	OS_NAME=ubuntu OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=24.04 docker buildx bake --no-cache ubuntu
	OS_NAME=ubuntu OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=24.04 ./test.sh

.PHONY: capms
capms: ubuntu
	KUBE_VERSION=1.32.11 \
	KUBE_APT_BRANCH=v1.32 \
	SEMVER_MAJOR_MINOR=1.32.11 \
	docker buildx bake --no-cache ubuntu-capms
	OS_NAME=capms-ubuntu OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=1.32.11 ./test.sh

.PHONY: firewall
firewall: binary
	mkdir -p "images/firewall/3.0-ubuntu"
	OS_NAME=firewall OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=3.0-ubuntu docker buildx bake --no-cache ubuntu-firewall
	OS_NAME=firewall OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=3.0-ubuntu ./test.sh

.PHONY: almalinux
almalinux: binary
	mkdir -p "images/almalinux/9"
	OS_NAME=almalinux OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=9 docker buildx bake --no-cache almalinux
	OS_NAME=almalinux OUTPUT_FOLDER="" SEMVER_MAJOR_MINOR=9 ./test.sh
