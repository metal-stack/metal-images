#!/usr/bin/env bash

set -ex

#MACHINE_TYPE=machine
#DOCKER_IMAGE=ghcr.io/metal-stack/ubuntu:24.04-stable

TAR=tar.tar
DISK=disk.raw
SIZE=4G
ROOTFS=./rootfs

sudo rm -rf ${TAR}

echo "Export ${DOCKER_IMAGE} to tar file"
time docker export "$(docker create "${DOCKER_IMAGE}")" > ${TAR}

echo "Extract tar file for a disk image"
truncate -s "$SIZE" "$DISK"
mkfs.ext4 -F -L rootfs "$DISK"
mkdir -p ${ROOTFS}
tune2fs -O ^orphan_file $DISK
sudo mount -o loop "$DISK" ${ROOTFS}
sudo tar xf ${TAR} -C ${ROOTFS}/

echo "Fix console for cloud hypervisor"

echo "Prepare chroot environment"
sudo mount -t proc proc "${ROOTFS}/proc"
sudo mount -t sysfs sys "${ROOTFS}/sys"
sudo mount -t efivarfs /sys/firmware/efi/efivars "${ROOTFS}/sys/firmware/efi/efivars"
sudo mount --bind /dev "${ROOTFS}/dev"

echo "Run /install-go in the chroot environment"
sudo chroot ${ROOTFS} /bin/bash -lc 'PATH=/sbin:$PATH MACHINE_TYPE='"${MACHINE_TYPE}"' INSTALL_FROM_CI=true /install-go'

echo "Extract kernel from os"
sudo ls -alh ${ROOTFS}/boot/
if sudo test -f ${ROOTFS}/boot/vmlinuz; then
    sudo cp ${ROOTFS}/boot/vmlinuz ./os-kernel
else
    sudo cp ${ROOTFS}/boot/vmlinuz* ./os-kernel
fi
if sudo test -f ${ROOTFS}/boot/initrd.img; then
    echo "nop"
elif test -f ${ROOTFS}/boot/initrd.img-*; then
    sudo cp ${ROOTFS}/boot/initrd.img-* ./initramfs
else
    sudo cp ${ROOTFS}/boot/initramfs* ./initramfs
fi

echo "Sync filesystem and umount"
sync

sudo umount ${ROOTFS}/proc
sudo umount ${ROOTFS}/sys/firmware/efi/efivars
sudo umount ${ROOTFS}/sys
sudo umount ${ROOTFS}/dev
sudo umount ${ROOTFS}
