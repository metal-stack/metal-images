#!/usr/bin/env bash

set -ex

TAR=tar.tar
DISK=disk.raw
SIZE=4G
ROOTFS=./rootfs

rm -rf ${TAR}

echo "Export ${DOCKER_IMAGE} to tar file"
time docker export "$(docker create "${DOCKER_IMAGE}")" > ${TAR}

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

echo "Run /install-go in the chroot environment"
chroot ${ROOTFS} /bin/bash -lc "PATH=/sbin:$PATH MACHINE_TYPE='${MACHINE_TYPE}' INSTALL_FROM_CI=true /install-go"

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

umount ${ROOTFS}/sys/firmware/efi/efivars
umount ${ROOTFS}/sys
umount ${ROOTFS}/proc
umount ${ROOTFS}/dev
umount ${ROOTFS}
