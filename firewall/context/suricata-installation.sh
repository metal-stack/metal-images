#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install suricata from suricata ppa repository"
    apt-get update --quiet
    apt-get install --yes --no-install-recommends software-properties-common
    add-apt-repository --yes ppa:oisf/suricata-stable
    apt-get update --quiet
    apt-get install --yes --no-install-recommends suricata
else
    echo "Debian - Install suricata from debian testing repository"
    # Note: suricata from main is still on 4.x, 5.x is in testing only
    echo "deb https://deb.debian.org/debian testing main" > /etc/apt/sources.list.d/testing.list
    rm -f /etc/logrotate.d/suricata
    apt-get update --quiet
    apt-get install --yes --no-install-recommends -t testing suricata suricata-update
fi

echo "Enable suricata service"
systemctl enable suricata