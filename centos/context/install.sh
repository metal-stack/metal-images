#!/usr/bin/env bash
set -e
# Workaround to fix empty path
export PATH="$PATH:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

source /etc/os-release
OS_NAME=${ID}
readonly BOOTLOADER_ID="metal-${OS_NAME}"

# Must be written here because during docker build this file is synthetic
echo -e "nameserver 1.1.1.1\nnameserver 1.0.0.1" > /etc/resolv.conf

readonly CONSOLE=$(yq r /etc/metal/install.yaml console)

# Serial port and speed are required by grub
readonly SERIAL_PORT=$(echo ${CONSOLE} | cut -d , -f 1 | tr -dc '0-9')
readonly SERIAL_SPEED=$(echo ${CONSOLE} | cut -d , -f 2 | cut -d n -f 1 | tr -dc '0-9')

export diskjson="/etc/metal/disk.json"

# figure out uuids of partitions to fill etc/fstab
readonly EFI_UUID=$(jq -r '.Partitions[] | select(.Label=="efi").Properties.UUID' "$diskjson")
readonly EFI_FS=$(jq -r '.Partitions[] | select(.Label=="efi").Filesystem' "$diskjson")
readonly EFI_MOUNTPOINT=/boot/efi
readonly ROOT_UUID=$(jq -r '.Partitions[] | select(.Label=="root").Properties.UUID' "$diskjson")
readonly ROOT_FS=$(jq -r '.Partitions[] | select(.Label=="root").Filesystem' "$diskjson")
readonly VARLIB_UUID=$(jq -r '.Partitions[] | select(.Label=="varlib").Properties.UUID' "$diskjson")
readonly VARLIB_FS=$(jq -r '.Partitions[] | select(.Label=="varlib").Filesystem' "$diskjson")

readonly CMDLINE="console=${CONSOLE} root=UUID=${ROOT_UUID} init=/usr/sbin/init net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295"

# only add /var/lib filesystem if created.
VARLIB=""
if [[ ! "${VARLIB_UUID}" = "" ]]
then
  VARLIB="UUID=\"${VARLIB_UUID}\" /var/lib ${VARLIB_FS} defaults 0 1"
fi

cat << EOM >/etc/fstab
UUID="${ROOT_UUID}" / ${ROOT_FS} defaults 0 1
${VARLIB}
UUID="${EFI_UUID}" ${EFI_MOUNTPOINT} ${EFI_FS} defaults 0 2
tmpfs /tmp tmpfs defaults,noatime,nosuid,nodev,noexec,mode=1777,size=512M 0 0
EOM

cat /etc/fstab

# create a user/pass (metal:metal) to enable login
# TODO move to Dockerfile
readonly user="metal"
readonly pass=$(yq r /etc/metal/install.yaml password)
readonly devmode=$(yq r /etc/metal/install.yaml devmode)
echo "creating user '$user'"
useradd --create-home --gid "wheel" --shell /bin/bash $user

echo "set password for $user to $pass expires after 1 day."
echo -e "$pass\n$pass" | passwd $user

if [ $devmode == "true" ]; then
    echo "password valid for 24h: user:$user password:$pass" >> /etc/issue
fi

# configure networking to setup interfaces and establish BGP/ EVPN sessions
# FIXME do not ignore any errors
/network.sh || true

# Take care: init must use systemd!
cat << EOM >/boot/grub2/grub.cfg
GRUB_DEFAULT=0
GRUB_TIMEOUT=5
GRUB_DISTRIBUTOR=${BOOTLOADER_ID}
GRUB_CMDLINE_LINUX_DEFAULT=""
GRUB_CMDLINE_LINUX="${CMDLINE}"
GRUB_TERMINAL=serial
GRUB_SERIAL_COMMAND="serial --speed=${SERIAL_SPEED} --unit=${SERIAL_PORT} --word=8"
EOM

if [ -d /sys/firmware/efi ]
then
    echo "System was booted with UEFI"
    # FIXME do not ignore any errors
    grub2-mkconfig -o /boot/efi/EFI/redhat/grub.cfg || true
else
    echo "System was booted with Bios"
    grub2-mkconfig -o /boot/grub2/grub.cfg || true
fi

# set sshpublickey
SSHDIR=~metal/.ssh
mkdir -p ${SSHDIR}
chown metal ${SSHDIR}
chmod 700 ${SSHDIR}
yq r /etc/metal/install.yaml sshpublickey > ${SSHDIR}/authorized_keys

echo "align directory permissions to OS defaults"
chmod 1777 /var/tmp
chmod 644 /etc/hosts

# execute ignition with userdata if present
if [ -e "/etc/metal/userdata" ]; then
    cd /etc/metal
    mv userdata config.ign
    echo "validate ignition config.ign"
    ignition-validate config.ign || true
    echo "execute ignition"
    ignition -oem file -stage files -log-to-stdout || true
    systemctl preset-all || true
    cd -
else
    echo "no userdata present"
fi

echo "write boot-info.yaml"

cd /boot
ln -s vmlinuz-* vmlinuz
ln -s initramfs-* initramfs.img
cd -
INITRD=$(readlink -f /boot/initramfs.img)
KERNEL=$(readlink -f /boot/vmlinuz)

cat <<REBOOT > /etc/metal/boot-info.yaml
---
initrd: ${INITRD}
cmdline: "${CMDLINE}"
kernel: ${KERNEL}
bootloader_id: ${BOOTLOADER_ID}
...
REBOOT
