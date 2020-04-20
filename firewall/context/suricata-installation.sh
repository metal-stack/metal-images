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
    echo "Debian - Install suricata from debian repository"
    rm -f /etc/logrotate.d/suricata
    apt-get update --quiet
    apt-get install --yes --no-install-recommends suricata suricata-update
fi

echo "Enable suricata service"
systemctl enable suricata