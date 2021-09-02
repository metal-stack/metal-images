#!/usr/bin/env bash
set -e
export DEBCONF_NONINTERACTIVE_SEEN="true"
export DEBIAN_FRONTEND="noninteractive"

# Workaround to fix empty path
export PATH="$PATH:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# shellcheck disable=SC1091
source /etc/os-release
OS_NAME=${ID}
readonly BOOTLOADER_ID="metal-${OS_NAME}"

# Must be written here because during docker build this file is synthetic
# FIXME enable systemd-resolved based approach again once we figured out why it does not work on the firewall
# most probably because the resolved must be running in the internet facing vrf.
# ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf
rm -f /etc/resolv.conf
cat << RESOLV >/etc/resolv.conf
nameserver 8.8.8.8
nameserver 8.8.4.4
RESOLV

export INSTALL_YAML="/etc/metal/install.yaml"
readonly CONSOLE=$(yq e '.console' "$INSTALL_YAML")

# Serial port and speed are required by grub
readonly SERIAL_PORT=$(echo "${CONSOLE}" | cut -d , -f 1 | tr -dc '0-9')
readonly SERIAL_SPEED=$(echo "${CONSOLE}" | cut -d , -f 2 | cut -d n -f 1 | tr -dc '0-9')

export DISK_JSON="/etc/metal/disk.json"

# figure out uuids of partitions to fill etc/fstab
readonly EFI_UUID=$(jq -r '.Partitions[] | select(.Label=="efi").Properties.UUID' "$DISK_JSON")
readonly EFI_FS=$(jq -r '.Partitions[] | select(.Label=="efi").Filesystem' "$DISK_JSON")
readonly EFI_MOUNTPOINT=/boot/efi
readonly ROOT_UUID=$(jq -r '.Partitions[] | select(.Label=="root").Properties.UUID' "$DISK_JSON")
readonly ROOT_FS=$(jq -r '.Partitions[] | select(.Label=="root").Filesystem' "$DISK_JSON")
readonly VARLIB_UUID=$(jq -r '.Partitions[] | select(.Label=="varlib").Properties.UUID' "$DISK_JSON")
readonly VARLIB_FS=$(jq -r '.Partitions[] | select(.Label=="varlib").Filesystem' "$DISK_JSON")

CMDLINE="console=${CONSOLE} root=UUID=${ROOT_UUID} init=/bin/systemd net.ifnames=0 biosdevname=0 nvme_core.io_timeout=4294967295 systemd.unified_cgroup_hierarchy=0"

if [[ $(mdadm --examine --scan) ]]; then
    echo "raid is configured"
    ROOT_DISK=$(blkid | grep $ROOT_UUID | awk -F':' '{ print $1 }')
    eval $(mdadm --detail --export $ROOT_DISK) && CMDLINE="$CMDLINE rd.md.uuid=${MD_UUID}" || true
fi

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
readonly user="metal"
readonly pass=$(yq e '.password' "$INSTALL_YAML")
readonly devmode=$(yq e '.devmode' "$INSTALL_YAML")
echo "creating user '$user'"
useradd --create-home --gid "sudo" --shell /bin/bash $user

echo "set password for $user to $pass expires after 1 day."
echo -e "$pass\n$pass" | passwd $user

if [ "$devmode" == "true" ]; then
    echo "password valid for 24h: user:$user password:$pass" >> /etc/issue
fi

# configure networking to setup interfaces and establish BGP/ EVPN sessions
/network.sh

# set sshpublickey
SSHDIR=~metal/.ssh
mkdir -p ${SSHDIR}
chown metal ${SSHDIR}
chmod 700 ${SSHDIR}
yq e '.sshpublickey' ${INSTALL_YAML} > ${SSHDIR}/authorized_keys

echo "align directory permissions to OS defaults"
chmod 1777 /var/tmp
chmod 644 /etc/hosts

# execute ignition/cloud-init userdata if present
if [ -e "/etc/metal/userdata" ]; then
    cd /etc/metal

    # check for cloud-init - cluster-api uses this header: https://github.com/kubernetes-sigs/cluster-api/blob/71ba9246f9cd2d718e81866cdc869c10fb90a9e4/bootstrap/kubeadm/internal/cloudinit/cloudinit.go#L33
    firstLine=$(sed '1q;d' ./userdata)
    secondLine=$(sed '2q;d' ./userdata)
    if [[ ${firstLine} == "#cloud-config" ]] || [[ "${secondLine}" == "#cloud-config" ]]; then
        echo "validate cloud-init userdata"
        cloud-init devel schema --config-file userdata || true
    else
        mv userdata config.ign
        echo "validate ignition config.ign"
        ignition-validate config.ign || true
        echo "execute ignition"
        ignition -oem file -stage files -log-to-stdout || true
    fi

    systemctl preset-all || true
    cd -
else
    echo "no userdata present"
fi

echo "write boot-info.yaml"
# ubuntu-19.04 and before placed a legacy /vmlinuz link to /boot/vmlinux-<actual-version>
if [ -e /vmlinuz ]; then
    INITRD=$(readlink -f /initrd.img)
    KERNEL=$(readlink -f /vmlinuz)
fi
# since then, the link is made in /boot
if [ -e /boot/vmlinuz ]; then
    INITRD=$(readlink -f /boot/initrd.img)
    KERNEL=$(readlink -f /boot/vmlinuz)
fi

cat << REBOOT > /etc/metal/boot-info.yaml
---
initrd: ${INITRD}
cmdline: "${CMDLINE}"
kernel: ${KERNEL}
bootloader_id: ${BOOTLOADER_ID}
...
REBOOT

# Take care: init must use systemd!
cat << EOM >/etc/default/grub
GRUB_DEFAULT=0
GRUB_TIMEOUT=5
GRUB_DISTRIBUTOR=$(lsb_release -i -s || echo "${BOOTLOADER_ID}")
GRUB_CMDLINE_LINUX_DEFAULT=""
GRUB_CMDLINE_LINUX="${CMDLINE}"
GRUB_TERMINAL=serial
GRUB_SERIAL_COMMAND="serial --speed=${SERIAL_SPEED} --unit=${SERIAL_PORT} --word=8"
EOM

if [ -d /sys/firmware/efi ]
then
    echo "System was booted with UEFI"
    if [[ $(mdadm --examine --scan) ]]; then
        mdadm --examine --scan > /etc/mdadm.conf
        echo "MAILADDR root" >>  /etc/mdadm.conf

        mkdir -p /var/lib/initramfs-tools
        update-initramfs -u

        EFI_DISKS=$(blkid | grep "PARTLABEL=\"efi\"" | awk -F':' '{ print $1 }')
        for EFI_DISK in $EFI_DISKS; do
            efibootmgr -c -d $EFI_DISK -p1 -l \\EFI\\${OS_NAME}\\grubx64.efi -L "${BOOTLOADER_ID}"
        done

        grub-install --target=x86_64-efi --efi-directory=${EFI_MOUNTPOINT} --boot-directory=/boot --bootloader-id="${BOOTLOADER_ID}" --no-nvram
    else
        grub-install --target=x86_64-efi --efi-directory=${EFI_MOUNTPOINT} --boot-directory=/boot --bootloader-id="${BOOTLOADER_ID}"
    fi

    update-grub2
    dpkg-reconfigure grub-efi-amd64-bin
else
    echo "System was booted with Bios which is unsupported"
    exit 1
fi

# Unset the machine-id (most importantly to avoid fixed MAC addresses of interfaces - otherwise packets will arrive at unintended places!)
echo "" > /etc/machine-id
echo "" > /var/lib/dbus/machine-id
