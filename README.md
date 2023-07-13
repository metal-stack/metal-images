# metal-images

This project builds operating system images usable for bare metal server provisioning with [metal-stack](https://metal-stack.io).
Every OS image is build from a Dockerfile, exported to a lz4 compressed tarball, and uploaded to <https://images.metal-stack.io/>.

For security scanning those images are also pushed to [quay.io/metalstack](https://quay.io/user/metalstack).

Further information about the image store is available at [IMAGE_STORE.md](./IMAGE_STORE.md).

Information about our initial architectural decisions may be found in [ARCHITECTURE.md](./ARCHITECTURE.md).

## Local development and integration testing

Before you can start developing changes for metal-images or even introduce new operating systems, you have to install the following tools:

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
```

For integration testing the images are started as [firecracker vm](https://firecracker-microvm.github.io/) with [weaveworks/ignite](https://github.com/weaveworks/ignite) and basic properties like interfaces to other metal-stack components, kernel parameters, internet reachability, DNS resolution etc. are checked with [goss](https://github.com/aelsabbahy/goss) in a GitHub action workflow. The integration tests are also executed when you build an image locally with.

## Supported Images

Currently these images are supported:

1. Debian 12
1. Ubuntu 22.04
1. Firewall 3.0-ubuntu (based on Ubuntu 22.04)

Unsupported images:

1. Centos 7.0

## Schedule

Builds from the master branch are scheduled on every sunday night at 1:10 o'clock to get fresh metal-images every week.

## How new images get usable in a metal-stack partition

Images are synced to partitions with a service that mirrors the public bucket and which runs on the management servers of partitions.

Released Images are accessible with under this image URL, where `20230710` here is the tag of this repository.

`http://images.metal-stack.io/metal-os/20230710/debian/12/img.tar.lz4`

Images built from the master branch are accessible with an image URL like this:

`http://images.metal-stack.io/metal-os/stable/debian/12/img.tar.lz4`

For other branches, the URL pattern is this:

`http://images.metal-stack.io/metal-os/pull_requests/${CI_COMMIT_REF_SLUG}/debian/12/img.tar.lz4`

Those URLs can be used to define an image at the metal-api.
