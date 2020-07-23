#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install kernel and systemd-timesyncd from ubuntu repository"
    apt-get install --yes linux-image-generic systemd-timesyncd
else
    echo "Debian - Install kernel and systemd-timesyncd from backports repository"
    # Note: for firewall images the backports kernel is a hard requirements because kernel >= 5.x is necessary for vxlan/evpn
    echo "deb https://deb.debian.org/debian ${VERSION_CODENAME}-backports main" > /etc/apt/sources.list.d/backports.list
    apt-get update --quiet
    apt-get install --yes -t buster-backports linux-image-amd64 systemd-timesyncd
fi

# Remove WIFI, netronome, v4l and liquidio firmware to save ~300MB image size
rm -rf /usr/lib/firmware/*wifi* \
    /usr/lib/firmware/netronome \
    /usr/lib/firmware/v4l* \
    /usr/lib/firmware/liquidio \
    /var/lib/apt/lists/*
