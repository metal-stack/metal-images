#!/usr/bin/env bash

set -ex

echo "Setting up bridge for VMs"
ip link add name vm-br0 type bridge || true
ip link set up dev vm-br0 || true
ip addr add 100.100.0.1/24 dev vm-br0 || true

echo "Allowing VMs to reach the internet"
# the VM's default route points to vm-br0, so the host needs to forward and masquerade its traffic;
# rules are inserted at the top of FORWARD because docker sets its policy to drop
sysctl -qw net.ipv4.ip_forward=1
iptables -t nat -C POSTROUTING -s 100.100.0.0/24 ! -o vm-br0 -j MASQUERADE 2>/dev/null || \
  iptables -t nat -A POSTROUTING -s 100.100.0.0/24 ! -o vm-br0 -j MASQUERADE
iptables -C FORWARD -i vm-br0 -j ACCEPT 2>/dev/null || \
  iptables -I FORWARD -i vm-br0 -j ACCEPT
iptables -C FORWARD -o vm-br0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || \
  iptables -I FORWARD -o vm-br0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT

echo "Setting up tap device for VM"
ip link del tap0 || true
ip tuntap add mode tap name tap0 || true
ip link set tap0 up || true
ip link set tap0 master vm-br0 || true

# kernels shipped with ubuntu based images allow for direct kernel boot without passing initrd to cloud-hypervisor
if [[ "${OS_NAME}" == "ubuntu" || "${OS_NAME}" == "capms-ubuntu" ]]; then
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
  wget -qO metal-kernel https://github.com/metal-stack/kernel/releases/latest/download/metal-kernel
fi

# the metal-kernel has no kvm-clock, so in nested virtualization it cannot calibrate
# the TSC (cloud-hypervisor provides no PIT/HPET) and hangs; pass the host's TSC frequency
CMDLINE="console=hvc0 root=/dev/vda rw init=/sbin/init"
if grep -qw hypervisor /proc/cpuinfo; then
  TSC_KHZ=${TSC_KHZ:-$(dmesg | sed -n 's/.*tsc: Detected \([0-9]*\)\.\([0-9]*\) MHz.*/\1\2/p' | head -n1)}
  # the boot messages are gone from the ring buffer on long running hosts, as guest the cpu MHz is the TSC frequency
  TSC_KHZ=${TSC_KHZ:-$(sed -n 's/^cpu MHz[[:space:]]*: \([0-9]*\)\.\([0-9]\{3\}\).*/\1\2/p' /proc/cpuinfo | head -n1)}
  if [ -n "${TSC_KHZ}" ]; then
    CMDLINE="${CMDLINE} tsc_early_khz=${TSC_KHZ}"
  fi
fi

echo "Running VM"
killall cloud-hypervisor || true
rm -f ./my.sock ./console.log || true
cloud-hypervisor ${INITRAMFS} \
  --api-socket my.sock \
  --kernel "./${KERNEL}" \
  --disk path="./disk.raw",image_type=raw \
  --cmdline "${CMDLINE}" \
  --cpus boot=1 \
  --serial off \
  --console file=./console.log \
  --memory size=1G \
  --net "tap=tap0,ip=100.100.0.3,mask=255.255.255.0,mac=00:03:00:11:11:01" &

sleep 5