#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu"
    apt install --yes linux-image-generic
else
    echo "Debian"
    # Note: for firewall images the backports kernel is a hard requirements because kernel >= 5.x is necessary for vxlan/evpn
    echo "deb https://deb.debian.org/debian ${VERSION_CODENAME}-backports main" > /etc/apt/sources.list.d/backports.list
    apt update --quiet
    apt install --yes -t buster-backports linux-image-amd64
fi