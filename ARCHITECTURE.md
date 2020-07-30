# Architectural Decisions

## Goals for creating OS images

* minimal set of installed packages
* customization has to be possible
* reproducible builds

## Our approach

* build an image based on the published docker images of OS vendors
* add necessary packages and services in a Dockerfile for the OS to provide:
  * sudo functionality
  * a SSH-Server
  * the FRR-Suite for BGP (the servers act as a BGP router/speaker)
  * yq for reading YAML-Files
* provide a `install.sh` file within the image that will be invoked by the `metal-hammer`
  * writes the `/etc/fstab` based on the UUIDs of the connected disks
  * setup UEFI-Boot
  * create an OS user with sudo rights
  * set the SSH public keys that are allowed to log into the system
  * sets the hostname
  * sets the network configuration of the server (IP-Addresses at loopback device, ASN for BGP)
  * sets the token used for phoning home to the metal-api

The `install.sh` has to be implemented for every OS. Between `install.sh` and the `metal-hammer` component exists this contract:

* the `metal-hammer` writes the file `/etc/metal/install.yaml` which contains data, that is not known during build time and can be customized by users of the `metal-api`

  ```yaml
  ---
  hostname: some
  ipaddress: 10.0.0.2/32
  asn: 420000001
  sshpublickeys: ""
  ```

* `install.sh` is expected to create the file `/etc/metal/boot-info.yaml` which contains data, that the `metal-hammer` will use for the `kexec-reboot` of the server.

  ```yaml
  ---
  initrd:
  cmdline:
  kernel:
  ```
