#!/usr/bin/env bash

set -ex

unmount () {
  umount ${ROOTFS}/sys/firmware/efi/efivars
  umount ${ROOTFS}/sys
  umount ${ROOTFS}/proc
  umount ${ROOTFS}/dev
  umount ${ROOTFS}
}

trap 'rc=$?; unmount; exit $rc' ERR

GOSS_VERSION=v0.4.7
GOSS_URL=https://github.com/goss-org/goss/releases/download/${GOSS_VERSION}/goss-linux-amd64
TAR=../img.tar
DISK=disk.raw
SIZE=4G
ROOTFS=./rootfs
OS_INSTALLER_VERSION=v0.3.1

# kernels shipped with ubuntu based images allow for direct kernel boot without passing initrd to cloud-hypervisor
if [[ "${OS_NAME}" == "ubuntu" || "${OS_NAME}" == "capms-ubuntu" ]]; then
  INITRAMFS=""
  KERNEL="os-kernel"
  MACHINE_TYPE="machine"
elif [[ "${OS_NAME}" == *"firewall" ]]; then
  INITRAMFS=""
  KERNEL="os-kernel"
  MACHINE_TYPE="firewall"
elif [[ "${OS_NAME}" == "debian" || "${OS_NAME}" == "debian-nvidia" ]]; then
  INITRAMFS=""
  KERNEL="metal-kernel"
  MACHINE_TYPE="machine"
else
  INITRAMFS="--initramfs ./initramfs"
  KERNEL="os-kernel"
  MACHINE_TYPE="machine"
fi

if [ "${KERNEL}" == "metal-kernel" ]; then
  echo "Downloading kernel"
  wget -qO metal-kernel https://github.com/metal-stack/kernel/releases/latest/download/metal-kernel
fi

echo "Extract tar file for a disk image"
truncate -s "$SIZE" "$DISK"
mkfs.ext4 -F -L rootfs "$DISK"
mkdir -p ${ROOTFS}
tune2fs -O ^orphan_file $DISK
mount -o loop "$DISK" ${ROOTFS}
tar xf ${TAR} -C ${ROOTFS}/

echo "Prepare chroot environment"
mount -t proc proc "${ROOTFS}/proc"
mount -t sysfs sys "${ROOTFS}/sys"
mount -t efivarfs /sys/firmware/efi/efivars "${ROOTFS}/sys/firmware/efi/efivars"
mount --bind /dev "${ROOTFS}/dev"

echo "Add sut-ctx"
rm -f "${ROOTFS}/etc/systemd/system/getty.target.wants/getty@tty1.service"
cp -rf sut-ctx/* "${ROOTFS}/"
mv "${ROOTFS}/etc/metal/${MACHINE_TYPE}.yaml" "${ROOTFS}/etc/metal/install.yaml"
mv "${ROOTFS}/etc/metal/userdata-${MACHINE_TYPE}.json" "${ROOTFS}/etc/metal/userdata"
wget -qO "${ROOTFS}/usr/local/bin/goss" "${GOSS_URL}"
chmod 755 "${ROOTFS}/usr/local/bin/goss"

echo "Run /install-go in the chroot environment"
docker create --name os-installer ghcr.io/metal-stack/os-installer:${OS_INSTALLER_VERSION}
docker cp os-installer:/os-installer ${ROOTFS}/os-installer
docker rm os-installer
chroot ${ROOTFS} /bin/bash -lc "PATH=/sbin:$PATH MACHINE_TYPE='${MACHINE_TYPE}' INSTALL_FROM_CI=true /os-installer"

echo "Extract kernel from os"
ls -alh ${ROOTFS}/boot/
if test -f ${ROOTFS}/boot/vmlinuz; then
    cp ${ROOTFS}/boot/vmlinuz ./os-kernel
else
    cp ${ROOTFS}/boot/vmlinuz* ./os-kernel
fi
if test -f ${ROOTFS}/boot/initrd.img; then
    echo "nop"
elif test -f ${ROOTFS}/boot/initrd.img-*; then
    cp ${ROOTFS}/boot/initrd.img-* ./initramfs
else
    cp ${ROOTFS}/boot/initramfs* ./initramfs
fi

echo "Sync filesystem and umount"
sync

unmount

echo "Setting up bridge for VMs"
ip link add name vm-br0 type bridge || true
ip link set up dev vm-br0 || true
ip addr add 100.100.0.1/24 dev vm-br0 || true

echo "Setting up tap device for VM"
ip link del tap0 || true
ip tuntap add mode tap name tap0 || true
ip link set tap0 up || true
ip link set tap0 master vm-br0 || true

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
  --net "tap=tap0,ip=100.100.0.3,mask=255.255.255.0,mac=00:03:00:11:11:01" \
  </dev/null >cloud-hypervisor.log 2>&1 &

sleep 5

echo "do machine test"
# FIXME remove this || true once we understand why goss tests fail on google dns.
ssh -F ./ssh/config -t machine <<"EOF" || true
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
