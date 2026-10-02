# Architectural Decisions

## Goals for creating OS images

* minimal set of installed packages
* customization has to be possible
* reproducible builds

## Our approach

* build an image based on the published docker images of OS vendors
* add necessary packages and services in a Dockerfile for the OS to provide:
  * sudo functionality
  * an SSH-Server
  * the [FRR-Suite](https://frrouting.org/) for BGP (the servers act as a BGP router/speaker)
  * [ignition](https://www.flatcar.org/docs/latest/provisioning/ignition/) for userdata execution
  * `mdadm` for raid fsl configuration
  * `lvm` for volume group fsl provisioning
  * `grub` for bootloader installation
  * `timesyncd` for NTP time synchronization
* these services will be configured by the [os-installer](https://github.com/metal-stack/os-installer) invoked during machine provisioning by the metal-hammer
* for out-of-tree custom images, build owners might want to provide a `/etc/metal/os-installer.yaml` file within the image that can provide special instructions for the os-installer
