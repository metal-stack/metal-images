#!/bin/bash

set -ex

echo "generating /etc/metal/disk.json"
UUID=$(lsblk -JO | jq -r '.blockdevices[0].uuid')
jq --arg uuid "${UUID}" '.Partitions[0].Properties.UUID = $uuid' /disk.json > /etc/metal/disk.json
cat /etc/metal/disk.json

if hash goss 2>/dev/null; then
    echo "goss is already installed"
else
    echo "installing goss"
    curl -L https://github.com/aelsabbahy/goss/releases/latest/download/goss-linux-amd64 -o /usr/local/bin/goss
    chmod +rx /usr/local/bin/goss
fi

echo "move input file for metal-networker to proper location"
mv /machine.yaml /etc/metal/install.yaml

echo "for idempotency: delete metal user before issuing install.sh"
userdel metal
