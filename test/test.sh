#!/bin/bash

set -ex

echo "copy input files and goss"
scp -o StrictHostKeyChecking=no -i ./key ./inputs/* "root@${IP}":/

echo "do machine test"
# somehow chrony@vrf104009 needs a double restart to work
ssh -o StrictHostKeyChecking=no -i ./key "root@${IP}" <<EOF
    MACHINE_TYPE=${MACHINE_TYPE} /prepare.sh
    /install.sh
    systemctl restart systemd-networkd
    systemctl restart chrony@vrf104009
    systemctl daemon-reload
    systemctl restart chrony@vrf104009
    systemctl restart frr
    systemctl restart nftables
    cd / && /goss.sh ${MACHINE_TYPE}
EOF
