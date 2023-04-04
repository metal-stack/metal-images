#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install suricata from suricata ppa repository"
    apt-get update --quiet
    apt-get install --yes --no-install-recommends software-properties-common
    add-apt-repository --yes ppa:oisf/suricata-stable
    apt-get update --quiet
    apt-get install --yes --no-install-recommends chrony suricata nftables || true
else
    echo "Debian - Install suricata"
    apt-get install --yes --no-install-recommends chrony suricata suricata-update nftables || true
    # remove testing list, otherwise doing update on the machine will show 100s of missing updates.
    rm -f /etc/apt/sources.list.d/testing.list
fi

echo "Move suricata logs to /var/lib/ partition"
mkdir -p /var/lib/suricata
mv /var/log/suricata /var/lib/suricata/log
ln -s /var/lib/suricata/log /var/log/suricata

echo "Enable suricata service and suricata-update timer"
systemctl enable suricata suricata-update.service suricata-update.timer
