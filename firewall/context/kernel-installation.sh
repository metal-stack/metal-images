#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu"
else
    echo "Debian"
    apt remove --yes linux-image-amd64
    echo "deb https://deb.debian.org/debian ${VERSION_CODENAME}-backports main" > /etc/apt/sources.list.d/backports.list
    apt update --quiet
    apt install --yes -t buster-backports linux-image-amd64
fi