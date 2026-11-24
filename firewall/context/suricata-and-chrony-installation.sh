#!/usr/bin/env bash
set -e
source /etc/os-release

# The build context already contains files which are shipped as conffiles by the packages
# installed below (e.g. /etc/logrotate.d/suricata, /etc/nftables.conf and /etc/suricata/update.yaml).
# dpkg treats those files as locally created and prompts for a decision. As the build is
# noninteractive the prompt hits EOF and dpkg aborts with
# "end of file on stdin at conffile prompt", leaving the affected files as *.dpkg-new and the
# package in state "iU" (unpacked but not configured). Keep our versions and never prompt.
DPKG_OPTS=(-o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold")

if [ "${ID}" = "ubuntu" ] ; then
    echo "Ubuntu - Install suricata"

    # Pre-Configure chrony instead of systemd-timesyncd because it is able to run in a VRF context without issues.
    # Final setup is left to metal-networker that knows the internet-facing VRF.
    # To succeed metal-networker enabling chrony it is important to provide the chrony unit template in advance.
    # Usually the generator creates that template but the generator is loaded only after system boot or at `systemctl daemon-reload` (cannot be run from Docker Context).
    # systemd-time-wait-sync.service is disabled because it sometimes does not start and blocks depending services like logrotate.
    # see https://github.com/systemd/systemd/issues/14061

    systemctl disable systemd-timesyncd
    apt-get update --quiet
    apt-get install --yes --no-install-recommends "${DPKG_OPTS[@]}" chrony suricata suricata-update nftables
else
    echo "Debian - Install suricata"
    apt-get install --yes --no-install-recommends "${DPKG_OPTS[@]}" chrony suricata suricata-update nftables
    # remove testing list, otherwise doing update on the machine will show 100s of missing updates.
    rm -f /etc/apt/sources.list.d/testing.list
fi

echo "Move suricata logs to /var/lib/ partition"
mkdir -p /var/lib/suricata
mv /var/log/suricata /var/lib/suricata/log
ln -s /var/lib/suricata/log /var/log/suricata

echo "Enable suricata service and suricata-update timer"
systemctl enable suricata suricata-update.service suricata-update.timer
