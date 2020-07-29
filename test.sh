#!/bin/bash

set -e

# example: sudo OS_NAME=ubuntu ./test.sh quay.io/metalstack/ubuntu:19.10
hash ignite 2>/dev/null || { echo >&2 "ignite not found please install from: https://github.com/weaveworks/ignite"; exit 1; }

export IMAGE="${1}"
export VM_NAME="vm-${OS_NAME}"
export MACHINE_TYPE="machine"
export KERNEL_IMAGE="weaveworks/ignite-kernel:5.4.43"

if [[ "$OS_NAME" == firewall* ]]; then
  export MACHINE_TYPE="firewall"
  # for firewalls we take the metal-stack kernel for nftables support by the kernel
  export KERNEL_IMAGE="metal-kernel"
fi

if [ "${KERNEL_IMAGE}" == "metal-kernel" ]; then
  echo "build metal-kernel oci"
  cd test && docker build . -t metal-kernel:latest && cd -

  echo "import metal-kernel image to ignite"
  sudo ignite kernel import --runtime=docker metal-kernel:latest
fi

echo "import image oci to ignite: ${IMAGE}"
sudo ignite stop "${VM_NAME}" || true
sudo ignite rm "${VM_NAME}" || true
sudo ignite image rm -f "${IMAGE}" || true
sudo ignite image import --runtime=docker "${IMAGE}"

echo "create ignite / firecracker vm"
chmod 0600 ./test/key
chmod 0644 ./test/key.pub
sudo ignite run "${IMAGE}" \
  --name "${VM_NAME}" \
  --kernel-image "${KERNEL_IMAGE}" \
  --size 4G \
  --ssh=./test/key.pub \
  --memory 1G --cpus 1 \
  --log-level debug

echo "determine ip address of vm"
IP=$(sudo ignite inspect vm "${VM_NAME}" -t "{{ .Status.IPAddresses }}")
export IP

while ! nc -z "${IP}" 22; do
  echo "ssh is not available yet"
  sleep 2
done

echo "ssh is available"
sleep 5

cd test
./test.sh
cd -
