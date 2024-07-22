#!/usr/bin/env bash

set -ex

echo "copy input files and goss"
scp -o StrictHostKeyChecking=no -i ./key ./inputs/* "root@${IP}":/

echo "do machine test"

# somehow chrony@vrf104009 needs a double restart to work
ssh -o StrictHostKeyChecking=no -i ./key "root@${IP}" <<EOF
    set -e
    MACHINE_TYPE=${MACHINE_TYPE} /prepare.sh
    # install go fails in a vm because grub-install will always fail.
    # When INSTALL_FROM_CI=true, some commands like grub-install are skipped in install.go
    export INSTALL_FROM_CI=true
    /install-go
    systemctl restart systemd-networkd
    systemctl restart frr

    if [ ${MACHINE_TYPE} == "firewall" ]; then
        systemctl restart chrony@vrf104009
        systemctl daemon-reload
        systemctl restart chrony@vrf104009
        systemctl restart nftables
    fi
    cd / && /goss.sh ${MACHINE_TYPE}
    cd / && [ "${CIS_VERSION}" != "" ] && CIS_VERSION=${CIS_VERSION} /cis-benchmark.sh
    echo "connection to ignite-vm completed"
EOF
