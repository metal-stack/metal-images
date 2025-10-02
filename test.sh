#!/usr/bin/env bash

set -e

# runs goss-tests against a running metal-image
# uses cloud-hypervisor to spin up a thin VM based on the docker image of a metal-image
# examples:
# - OS_NAME=ubuntu ./test.sh ghcr.io/metal-stack/ubuntu:24.04-stable
# - OS_NAME=debian ./test.sh ghcr.io/metal-stack/debian:12-stable
# - OS_NAME=debian-nvidia ./test.sh ghcr.io/metal-stack/debian-nvidia:12-stable
# - OS_NAME=debian CIS_VERSION=v4.1-4 ./test.sh ghcr.io/metal-stack/debian:12-stable
# - OS_NAME=almalinux ./test.sh ghcr.io/metal-stack/almalinux:9-stable
# - OS_NAME=firewall ./test.sh ghcr.io/metal-stack/firewall:3.0-ubuntu-stable

hash cloud-hypervisor 2>/dev/null || { echo >&2 "cloud-hypervisor not found please install from: https://github.com/cloud-hypervisor/cloud-hypervisor"; exit 1; }

export MACHINE_TYPE="machine"
if [ "${OS_NAME}" == "firewall" ]; then
    export MACHINE_TYPE="firewall"
fi

export DOCKER_IMAGE="${1}"
echo "Testing ${MACHINE_TYPE} ${DOCKER_IMAGE}"
echo "delete cached images"
docker rmi "$DOCKER_IMAGE" || true
chmod 0600 ./test/files/key
chmod 0644 ./test/files/key.pub

cd ./test
./00_create_disk.sh
./01_start_vm.sh
./02_run_tests_in_vm.sh

