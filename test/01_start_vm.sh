#!/usr/bin/env bash

set -ex

echo "Setting up bridge for VMs"
ip link add name vm-br0 type bridge || true
ip link set up dev vm-br0 || true
ip addr add 100.100.0.1/24 dev vm-br0 || true

echo "Setting up tap device for VM"
ip tuntap add mode tap name tap0 || true
ip link set tap0 up || true
ip link set tap0 master vm-br0 || true

# kernels shipped with ubuntu based images allow for direct kernel boot without passing initrd to cloud-hypervisor
if [[ "${OS_NAME}" == "ubuntu" ]]; then
  INITRAMFS=""
  KERNEL="os-kernel"
elif [[ "${OS_NAME}" == *"firewall" ]]; then
  INITRAMFS=""
  KERNEL="os-kernel"
elif [[ "${OS_NAME}" == "debian" || "${OS_NAME}" == "debian-nvidia" ]]; then
  INITRAMFS=""
  KERNEL="metal-kernel"
else
  INITRAMFS="--initramfs ./initramfs"
  KERNEL="os-kernel"
fi

if [ "${KERNEL}" == "metal-kernel" ]; then
  echo "Downloading kernel"
  wget -O metal-kernel https://github.com/metal-stack/kernel/releases/latest/download/metal-kernel
fi

echo "Running VM"
killall cloud-hypervisor || true
rm -f ./my.sock || true
cloud-hypervisor ${INITRAMFS} \
  --api-socket my.sock \
  --kernel "./${KERNEL}" \
  --disk path="./disk.raw" \
  --cmdline "console=hvc0 root=/dev/vda rw init=/sbin/init" \
  --cpus boot=1 \
  --serial off \
  --console off \
  --memory size=1G \
  --net "tap=tap0,mac=00:03:00:11:11:01" &

sleep 3