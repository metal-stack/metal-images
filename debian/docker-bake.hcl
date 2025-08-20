group "all" {
    targets = ["debian-12","ubuntu-2404"]
}

target "_common" {
    args = {
        IGNITION_BRANCH = "v0.36.2"
        GOLLDPD_VERSION = "v0.4.9"
        CRI_VERSION = "v1.33.0"
        ICE_VERSION = "1.14.13"
        ICE_PKG_VERSION = "1.3.36.0"
    }
}

target "debian-12" {
    inherits = ["_common"]
    args = {
        BASE_OS_NAME = "debian"
        BASE_OS_VERSION = "bookworm"
        DOCKER_APT_OS = "debian"
        DOCKER_APT_CHANNEL ="bookworm"
        FRR_VERSION ="frr-10"
        FRR_VERSION_DETAIL ="10.4.1-0~deb12u1"
        FRR_APT_CHANNEL ="bookworm"
        SEMVER_MAJOR_MINOR = 12
      # see https://packages.debian.org/bookworm/kernel/ for available versions
        KERNEL_VERSION = "6.1.0-38"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/debian:12"]
    context = "."
}

target "ubuntu-2404" {
    inherits = ["_common"]
    args = {
        BASE_OS_NAME = "ubuntu"
        BASE_OS_VERSION = "24.04"
        DOCKER_APT_OS = "ubuntu"
        DOCKER_APT_CHANNEL ="noble"
        FRR_VERSION ="frr-10"
        FRR_VERSION_DETAIL ="10.4.1-0~ubuntu24.04.1"
        FRR_APT_CHANNEL ="noble"
        SEMVER_MAJOR_MINOR = 24.04
        # see https://kernel.ubuntu.com/mainline for available versions
        UBUNTU_MAINLINE_KERNEL_VERSION = "v6.12.42"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/ubuntu:24.04"]
    context = "."
}

