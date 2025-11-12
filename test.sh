#!/usr/bin/env bash

set -ex

# runs goss-tests against a running metal-image
# uses cloud-hypervisor to spin up a thin VM based on the docker image of a metal-image
# examples:
# WORKS						OS_NAME=ubuntu ./test.sh ghcr.io/metal-stack/ubuntu:24.04-stable
# WORKS 					OS_NAME=debian CIS_VERSION=v4.1-4 ./test.sh ghcr.io/metal-stack/debian:12-stable
# WORKS_WITH_METAL_KERNEL   OS_NAME=debian-nvidia ./test.sh ghcr.io/metal-stack/debian-nvidia:12-stable
# WORKS_WITH_METAL_KERNEL   OS_NAME=debian ./test.sh ghcr.io/metal-stack/debian:12-stable
# WORKS					    OS_NAME=firewall ./test.sh ghcr.io/metal-stack/firewall:3.0-ubuntu-stable
# WORKS                     OS_NAME=almalinux ./test.sh ghcr.io/metal-stack/almalinux:9-stable

export MACHINE_TYPE="machine"
if [[ "${OS_NAME}" == *"firewall" ]]; then
    export MACHINE_TYPE="firewall"
fi

echo "Testing ${MACHINE_TYPE} ${DOCKER_IMAGE}"
chmod 0600 ./test/files/key
chmod 0644 ./test/files/key.pub

cd ./test
sudo MACHINE_TYPE="${MACHINE_TYPE}" \
  OS_NAME="${OS_NAME}" \
  OUTPUT_FOLDER="${OUTPUT_FOLDER}" \
  SEMVER_MAJOR_MINOR="${SEMVER_MAJOR_MINOR}" \
  ./00_create_disk.sh
sudo OS_NAME="${OS_NAME}" ./01_start_vm.sh
sudo OS_NAME="${OS_NAME}" MACHINE_TYPE="${MACHINE_TYPE}" CIS_VERSION="${CIS_VERSION}" ./02_run_tests_in_vm.sh
