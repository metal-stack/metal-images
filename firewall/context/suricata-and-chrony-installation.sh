#!/usr/bin/env bash
set -e
source /etc/os-release

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install suricata from suricata ppa repository"

    # Pre-Configure chrony instead of systemd-timesyncd because it is able to run in a VRF context without issues.
    # Final setup is left to metal-networker that knows the internet-facing VRF.
    # To succeed metal-networker enabling chrony it is important to provide the chrony unit template in advance.
    # Usually the generator creates that template but the generator is loaded only after system boot or at `systemctl daemon-reload` (cannot be run from Docker Context).
    # systemd-time-wait-sync.service is disabled because it sometimes does not start and blocks depending services like logrotate.
    # see https://github.com/systemd/systemd/issues/14061

    systemctl disable systemd-timesyncd

    apt-get update --quiet
    apt-get install --yes --no-install-recommends software-properties-common
    add-apt-repository --yes ppa:oisf/suricata-stable
    apt-get remove --yes polkitd software-properties-common
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
