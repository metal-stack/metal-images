#!/bin/bash

set -ex

echo "generating /etc/metal/disk.json"
UUID=$(lsblk -JO | jq -r '.blockdevices[0].uuid')
jq --arg uuid "${UUID}" '.Partitions[0].Properties.UUID = $uuid' /disk.json > /etc/metal/disk.json
cat /etc/metal/disk.json

echo "move input file for metal-networker to proper location"
mv "/${MACHINE_TYPE}.yaml" /etc/metal/install.yaml

echo "place userdata"
mv "/userdata-${MACHINE_TYPE}.json" /etc/metal/userdata

echo "for idempotency: delete metal user before issuing install.sh"
userdel metal