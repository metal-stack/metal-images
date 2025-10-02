#!/usr/bin/env bash

set -e

#MACHINE_TYPE=machine
#DOCKER_IMAGE=ghcr.io/metal-stack/ubuntu:24.04-stable

TAR=tar.tar
DISK=disk.raw
SIZE=4G
ROOTFS=/mnt/chroot

sudo rm -rf ${TAR}

echo "Export ${DOCKER_IMAGE} to tar file"
time docker export "$(docker create "${DOCKER_IMAGE}")" > ${TAR}

echo "Extract tar file for a disk image"
truncate -s "$SIZE" "$DISK"
mkfs.ext4 -F -L rootfs "$DISK"
sudo mkdir -p ${ROOTFS}
sudo mount -o loop "$DISK" ${ROOTFS}
sudo tar xf ${TAR} -C ${ROOTFS}/

echo "Fix console for cloud hypervisor"
sudo cp ./files/serial-getty@hvc0.service ${ROOTFS}/etc/systemd/system/serial-getty@hvc0.service
# Basic inittab: spawn a getty on Cloud-Hypervisor’s console (hvc0)
cat <<'EOF' | sudo tee ${ROOTFS}/etc/inittab
::sysinit:/bin/mount -t proc proc /proc
::sysinit:/bin/mount -t sysfs sysfs /sys
::sysinit:/bin/mount -o remount,rw /
::respawn:/sbin/getty -L hvc0 115200 vt100
::ctrlaltdel:/bin/umount -a -r
::shutdown:/bin/umount -a -r
EOF
sudo unlink ${ROOTFS}/etc/systemd/system/getty.target.wants/getty@tty1.service
sudo ln -s /etc/systemd/system/serial-getty@hvc0.service ${ROOTFS}/etc/systemd/system/getty.target.wants/serial-getty@hvc0.service

echo "Copy userdata and install.yaml to proper places"
sudo cp ./files/${MACHINE_TYPE}.yaml ${ROOTFS}/etc/metal/install.yaml
sudo cp ./files/userdata-${MACHINE_TYPE}.json ${ROOTFS}/etc/metal/userdata

echo "Prepare chroot environment"
sudo mount -t proc proc "${ROOTFS}/proc"
sudo mount -t sysfs sys "${ROOTFS}/sys"
sudo mount -t efivarfs /sys/firmware/efi/efivars "${ROOTFS}/sys/firmware/efi/efivars"
sudo mount --bind /dev "${ROOTFS}/dev"

echo "Run /install-go in the chroot environment"
sudo chroot ${ROOTFS} /bin/bash -lc 'PATH=/sbin:$PATH INSTALL_FROM_CI=true /install-go'

echo "Sync filesystem and umount"
sudo sync
sudo umount ${ROOTFS}/proc
sudo umount ${ROOTFS}/sys/firmware/efi/efivars
sudo umount ${ROOTFS}/sys
sudo umount ${ROOTFS}/dev
sudo umount ${ROOTFS}
