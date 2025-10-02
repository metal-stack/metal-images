#!/bin/bash

set -e

echo "Setting up bridge for VMs"
sudo ip link add name vm-br0 type bridge || true
sudo ip link set up dev vm-br0 || true
sudo ip addr add 100.100.0.1/24 dev vm-br0 || true

echo "Setting up tap device for VM"
sudo ip tuntap add mode tap name tap0 || true
sudo ip link set tap0 up || true
sudo ip link set tap0 master vm-br0 || true

echo "Downloading kernel"
wget -O metal-kernel https://github.com/metal-stack/kernel/releases/latest/download/metal-kernel

echo "Running VM"
sudo cloud-hypervisor \
  --api-socket my.sock \
  --kernel "./metal-kernel" \
  --disk path="./disk.raw" \
  --cmdline "console=hvc0 root=/dev/vda rw init=/sbin/init ip=link-local" \
  --console off \
  --cpus boot=1 \
  --memory size=1G \
  --net "tap=tap0,mac=00:03:00:11:11:01" &
