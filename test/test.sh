#!/bin/bash

set -ex

echo "copy input files and goss"
scp -o StrictHostKeyChecking=no -i ../key ./inputs/* "root@${IP}":/

echo "do machine test"
ssh -o StrictHostKeyChecking=no -i ../key "root@${IP}" <<EOF
    MACHINE_TYPE=${MACHINE_TYPE} /prepare.sh
    /install.sh
    systemctl restart systemd-networkd
    systemctl restart frr
    systemctl restart nftables
    cd / && /goss.sh ${MACHINE_TYPE}
EOF
