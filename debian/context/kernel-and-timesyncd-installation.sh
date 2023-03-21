#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install kernel"
    # Download mainline kernel packages, kernel up to 5.13 available in ubuntu 20.04 and 22.04 have a broken NAT implementation.
    cd /tmp
    wget --no-directories \
         --no-parent \
         --accept-regex generic \
         --recursive \
         --execute robots=off \
        https://kernel.ubuntu.com/~kernel-ppa/mainline/${UBUNTU_MAINLINE_KERNEL_VERSION}/amd64/

    apt-get install --yes \
        /tmp/linux-image* \
        /tmp/linux-modules* \
        intel-microcode
else
    echo "Debian - Install kernel"

    echo "deb http://deb.debian.org/debian bullseye main contrib non-free" > /etc/apt/sources.list
    echo "deb http://security.debian.org/debian-security bullseye-security main contrib non-free" >> /etc/apt/sources.list
    echo "deb http://deb.debian.org/debian bullseye-updates main contrib non-free" >> /etc/apt/sources.list
    echo "deb http://deb.debian.org/debian bullseye-backports main" >> /etc/apt/sources.list

    apt update && apt install -y intel-microcode "linux-image-${KERNEL_VERSION}-amd64-unsigned"
fi

# Remove WIFI, netronome, v4l and liquidio firmware to save ~300MB image size
rm -rf /usr/lib/firmware/*wifi* \
    /usr/lib/firmware/netronome \
    /usr/lib/firmware/v4l* \
    /usr/lib/firmware/liquidio \
    /var/lib/apt/lists/* \
    /tmp/*
