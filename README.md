# metal-images

This project builds operating system images usable for bare metal server provisioning with [metal-stack](https://metal-stack.io).
Every OS image is built from a Dockerfile, exported to a lz4 compressed tarball, and uploaded to <https://images.metal-stack.io/>.

Further information about the image store is available at [IMAGE_STORE.md](./IMAGE_STORE.md).

Information about our initial architectural decisions may be found in [ARCHITECTURE.md](./ARCHITECTURE.md).

## Supported Images

Currently these images are supported:

1. Debian 12
1. Ubuntu 22.04
1. Firewall 3.0-ubuntu (based on Ubuntu 22.04)
1. Nvidia (based on Debian 12)

## Unsupported Images

We also build images that we require for special purposes but do not officially support. Only use at own risk.

1. CentOS 7
1. Almalinux 9

### GPU Support

With the nvidia image a worker has GPU support. Please check our official documentation on [docs.metal-stack.io](https://docs.metal-stack.io/stable/overview/gpu-support/) on how to get this running on Kubernetes.

## How new images become usable in a metal-stack partition

Images are synced to partitions with a service called [metal-image-cache-sync](https://github.com/metal-stack/metal-image-cache-sync). The service mirrors the public os images to the management servers and transparently serves the metal-images inside a partition.

Released images are tagged with the release date and accessible on the following image URL pattern:

`https://images.metal-stack.io/metal-os/20230710/debian/12/img.tar.lz4`

Images built from the master branch are accessible on an image URL like this:

`https://images.metal-stack.io/metal-os/stable/debian/12/img.tar.lz4`

For other branches, the URL pattern is this:

`https://images.metal-stack.io/metal-os/pull_requests/${CI_COMMIT_REF_SLUG}/debian/12/img.tar.lz4`

Those URLs can be used to define an image at the metal-api.

## Local development and integration testing

Please also consider our documentation on docs.metal-stack.io on [Build Your Own Images](https://docs.metal-stack.io/stable/overview/os/#Building-Your-Own-Images) to check for the contract an os image is expected to fulfill.

Before you can start developing changes for metal-images or even introduce new operating systems, you should install the following tools:

- **docker**: for sure
- **kvm**: hypervisor used for integration tests
- **lz4**: to compress tarballs
- **[docker-make](https://github.com/fi-ts/docker-make)**: this is a helper tool to define docker builds declaratively with YAML
- **[weaveworks/ignite](https://github.com/weaveworks/ignite)**: handles [firecracker vms](https://firecracker-microvm.github.io/) to spin up a metal-image virtually as VM

You can build metal-images like that:

```bash
# for debian images
make debian

# for ubuntu images
make ubuntu

# for firewall images
make firewall

# for centos images
make centos

# for nvidia images
make nvidia

# for almalinux images
make almalinux
```

For integration testing the images are started as [firecracker vm](https://firecracker-microvm.github.io/) with [weaveworks/ignite](https://github.com/weaveworks/ignite) and basic properties like interfaces to other metal-stack components, kernel parameters, internet reachability, DNS resolution etc. are checked with [goss](https://github.com/aelsabbahy/goss) in a GitHub action workflow. The integration tests are also executed when you build an image locally with.

### Debugging Image Provisioning

In some cases it might be necessary to manually figure out commands for provisioning a machine image. To do this on a real server environment, it is possible to hook into the metal-hammer through the machine's serial console.

At any time you can interrupt the metal-hammer by sending a keyboard interrupt. The metal-hammer does a short break before booting into the operating system kernel, which is a good moment for sending the interrupt.

To prevent the machine from rebooting, you should issue the following command immediately:

```bash
while true; do echo "1" > /dev/watchdog && sleep 55; done &
```

If you want to enter the operating system through `chroot`, you need to remount some file systems that were mounted by the metal-hammer during provisioning:

```bash
# the mount points also depend on the file system layout of the machine, so please only take this as an example:
mount /dev/sda2 /rootfs
mount -t vfat /dev/sda1 /rootfs/boot/efi
mount -t proc /proc /rootfs/proc
mount -t sysfs /sys /rootfs/sys
mount -t efivarfs /sys/firmware/efi/efivars /rootfs/sys/firmware/efi/efivars
mount -t devtmpfs /dev /rootfs/dev
```

Finally, you can then enter the provisioned OS image.

```bash
chroot /rootfs

# maybe you can mount further file systems here, which was not possible in the u-root environment of the metal-hammer
vgchange -ay
mount /dev/csi-lvm/varlib /var/lib/
```

Keep in mind that you are still running on the metal-hammer kernel, which is different from the kernel that will be run in the operating system after provisioning. For further information on the metal-stack machine provisioning sequence, check out documentation on [docs.metal-stack.io](https://docs.metal-stack.io/stable/overview/architecture/#Machine-Provisioning-Sequence). The kernel used by the metal-hammer is built by our project on the [kernel repository](https://github.com/metal-stack/kernel).
