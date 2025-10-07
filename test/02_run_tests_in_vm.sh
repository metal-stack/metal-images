#!/usr/bin/env bash

set -e

echo "do machine test"
ssh -F ./ssh/config -t machine <<EOF
    set -ex
    sudo ip addr add 100.100.0.2/24 dev lan0 || true
    sudo ip route add default via 100.100.0.1 dev lan0 || true
    cd /home/metal/inputs && sudo OS_NAME=${OS_NAME} ./goss.sh ${MACHINE_TYPE}
    cd /home/metal/inputs && [ "${CIS_VERSION}" != "" ] && sudo CIS_VERSION=${CIS_VERSION} ./cis-benchmark.sh
    echo "test completed"
EOF

echo "Terminating cloud-hypervisor processes"
killall cloud-hypervisor
rm -f ./my.sock