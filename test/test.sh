#!/bin/bash

set -ex

echo "copy input files and goss"
scp -o StrictHostKeyChecking=no -i key ./inputs/* root@${IP}:/

echo "do machine test"
ssh -o StrictHostKeyChecking=no -i key root@${IP} <<EOF
    /prepare.sh
    /install.sh
    systemctl restart frr
    systemctl restart networking
    cd / && goss validate -f documentation --color
EOF
