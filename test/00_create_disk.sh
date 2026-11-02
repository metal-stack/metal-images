#!/usr/bin/env bash

set -x

unmount () {
  if mountpoint -q ${ROOTFS}/sys/firmware/efi/efivars; then
    umount ${ROOTFS}/sys/firmware/efi/efivars
  fi
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
# efivarfs is only available on hosts booted via UEFI
if [ -d /sys/firmware/efi/efivars ]; then
  mount -t efivarfs /sys/firmware/efi/efivars "${ROOTFS}/sys/firmware/efi/efivars"
fi
mount --bind /dev "${ROOTFS}/dev"

echo "Add sut-ctx"
rm -f "${ROOTFS}/etc/systemd/system/getty.target.wants/getty@tty1.service"
cp -rf sut-ctx/* "${ROOTFS}/"
mv "${ROOTFS}/etc/metal/${MACHINE_TYPE}.yaml" "${ROOTFS}/etc/metal/install.yaml"
mv "${ROOTFS}/etc/metal/userdata-${MACHINE_TYPE}.json" "${ROOTFS}/etc/metal/userdata"
if [ "${MACHINE_TYPE}" == "firewall" ]; then
  mv "${ROOTFS}/etc/metal/firewall-allocation.yaml" "${ROOTFS}/etc/metal/machine-allocation.yaml"
else
  rm -f "${ROOTFS}/etc/metal/firewall-allocation.yaml"
fi
wget -qO "${ROOTFS}/usr/local/bin/goss" "${GOSS_URL}"
chmod 755 "${ROOTFS}/usr/local/bin/goss"
printf 'OS_NAME=%s\nMACHINE_TYPE=%s\n' "${OS_NAME}" "${MACHINE_TYPE}" > "${ROOTFS}/etc/metal/test.env"

echo "Run /os-installer in the chroot environment"
docker create --name os-installer ghcr.io/metal-stack/os-installer:${OS_INSTALLER_VERSION}
docker cp os-installer:/os-installer ${ROOTFS}/os-installer
docker rm os-installer
# the image sets umask 077 for login shells (CIS 5.4.4); os-installer needs a normal
# umask, otherwise e.g. systemd-networkd cannot read the generated .network files
chroot ${ROOTFS} /bin/bash -lc "umask 022; PATH=/sbin:$PATH MACHINE_TYPE='${MACHINE_TYPE}' INSTALL_FROM_CI=true /os-installer"

# enabled after the installation, os-installer runs systemctl preset-all on almalinux which removes it again;
# systemctl is not available in the chroot of every image
ln -sf /etc/systemd/system/goss.service "${ROOTFS}/etc/systemd/system/multi-user.target.wants/goss.service"

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
