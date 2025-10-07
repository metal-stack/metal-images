#!/usr/bin/env bash

# shellcheck disable=SC1091
source /etc/os-release

set -e

export GOSS_VERSION=v0.4.7
export OS=$ID
export MACHINE_TYPE=$1
export ROUTER_ID=10.1.0.1
export ASN=4200003073
export PUB_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIhLhxQwOgAaJmqZ9njCdfNw2+LTm24CwUTay6ZYlJBQ metal-images"
export PATH=$PATH:/usr/local/bin

goss validate -f documentation --color --retry-timeout 150s --sleep 10s