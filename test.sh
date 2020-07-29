#!/bin/bash

set -e

# example: sudo OS_NAME=ubuntu ./test.sh quay.io/metalstack/ubuntu:19.10
hash footloose 2>/dev/null || { echo >&2 "footloose not found please install from: https://github.com/weaveworks/footloose"; exit 1; }

hash ignite 2>/dev/null || { echo >&2 "ignite not found please install from: https://github.com/weaveworks/ignite"; exit 1; }

export IMAGE="${1}"
export VM="imagevm-${OS_NAME}0"
export MACHINE_TYPE="machine"

if [ "$OS_NAME" == "firewall" ]; then
  export MACHINE_TYPE="firewall"
fi

echo "build metal-kernel oci"
cd test && docker build . -t metal-kernel && cd -

echo "import metal-kernel oci to ignite"
sudo ignite kernel import --runtime=docker metal-kernel

echo "import image oci to ignite: ${IMAGE}"
sudo ignite stop "${VM}" || true
sudo ignite rm "${VM}" || true
sudo ignite image rm -f "${IMAGE}" || true
sudo ignite image import --runtime=docker "${IMAGE}"

echo "generate footloose config"
FOOTLOOSE_CFG="footloose.${OS_NAME}.yaml"
envsubst < test/footloose.yaml.tpl > "${FOOTLOOSE_CFG}"

echo "creat ignite / firecracker vm with footloose"
sudo footloose create -c "${FOOTLOOSE_CFG}"
sudo chown "$(id -u):$(id -g)" key key.pub
chmod 0600 key
chmod 0644 key.pub

echo "determine ip address of vm"
export IP=$(sudo ignite inspect vm "${VM}" -t "{{ .Status.IPAddresses }}")

while ! nc -z "${IP}" 22; do
  echo "ssh is not available yet"
  sleep 2
done

echo "ssh is available"
sleep 5

cd test
./test.sh
cd -

sudo footloose delete -c "${FOOTLOOSE_CFG}"
sudo ignite image rm "${IMAGE}"