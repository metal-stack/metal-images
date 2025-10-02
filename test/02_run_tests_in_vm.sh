#!/bin/bash

set -e

echo "copy input files"
scp -6 -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i ./files/key ./inputs/* "metal@[fe80::203:ff:fe11:1101%vm-br0]":/home/metal/

echo "do machine test"
# somehow chrony@vrf104009 needs a double restart to work
ssh -6 -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i ./files/key "metal@fe80::203:ff:fe11:1101%vm-br0" <<EOF
    set -e
    if [ ${MACHINE_TYPE} == "firewall" ]; then
        sudo systemctl restart chrony@vrf104009
        sudo systemctl daemon-reload
        sudo systemctl restart chrony@vrf104009
        sudo systemctl restart nftables
    fi
    sudo ip addr add 100.100.0.2/24 dev lan0 || true
    sudo ip route add default via 100.100.0.1 dev lan0 || true
    cd /home/metal && sudo OS_NAME=${OS_NAME} ./goss.sh ${MACHINE_TYPE}
    cd /home/metal && [ "${CIS_VERSION}" != "" ] && CIS_VERSION=${CIS_VERSION} sudo ./cis-benchmark.sh
    echo "test completed"
EOF

sudo killall cloud-hypervisor
