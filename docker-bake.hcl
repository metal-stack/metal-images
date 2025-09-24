target "_common" {
    attest = [
        "type=provenance,mode=max",
        "type=sbom",
    ]
    # no-cache = true
    output = [
        "type=registry",
    ]
}

target "_common_args" {
    args = {
        IGNITION_BRANCH = "v0.36.2"
        GOLLDPD_VERSION = "v0.4.10"
        CRI_VERSION = "v1.34.0"
        ICE_VERSION = "1.14.13"
        ICE_PKG_VERSION = "1.3.36.0"
    }
}

# declare empty var in order to be able to get value from shell
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}
variable "SEMVER" {}

target "almalinux" {
    inherits = ["_common", "_common_args"]
    dockerfile = "./almalinux/Dockerfile"
    contexts = {
        ctx = "./almalinux/context"
    }
    args = {
        BASE_OS_VERSION = 9
        FRR_VERSION="frr-stable"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    tags = ["ghcr.io/metal-stack/almalinux:${SEMVER}"]
}

target "debian" {
    inherits = ["_common", "_common_args"]
    dockerfile = "./debian/Dockerfile"
    contexts = {
        cloud-init = "./debian/cloud-init"
        ctx = "./debian/context"
    }
    args = {
        BASE_OS_NAME = "debian"
        BASE_OS_VERSION = "bookworm"
        DOCKER_APT_OS = "debian"
        DOCKER_APT_CHANNEL ="bookworm"
        FRR_VERSION ="frr-10"
        FRR_VERSION_DETAIL ="10.4.1-0~deb12u1"
        FRR_APT_CHANNEL ="bookworm"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
      # see https://packages.debian.org/bookworm/kernel/ for available versions
        KERNEL_VERSION = "6.1.0-39"
    }
    tags = ["ghcr.io/metal-stack/debian:${SEMVER}"]
}

target "debian-firewall" {
    inherits = ["_common"]
    dockerfile = "./firewall/Dockerfile"
    contexts = {
        baseapp = "target:debian"
        ctx = "./firewall/context"
    }
    args = {
        BASE_OS_VERSION = 12
        BASE_OS_NAME = "ghcr.io/metal-stack/debian"
        SEMVER_MAJOR_MINOR = "3.0"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    tags = ["ghcr.io/metal-stack/firewall:3.0${SEMVER_PATCH}"]
}

target "debian-nvidia" {
    inherits = ["_common"]
    dockerfile = "./debian-nvidia/Dockerfile"
    contexts = {
        baseapp = "target:debian"
        ctx = "./debian-nvidia/context"
    }
    args = {
        BASE_OS_VERSION = 12
        BASE_OS_NAME = "ghcr.io/metal-stack/debian"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    tags = ["ghcr.io/metal-stack/debian-nvidia:${SEMVER}"]
}

target "ubuntu" {
    inherits = ["_common", "_common_args"]
    dockerfile = "./debian/Dockerfile"
    contexts = {
        cloud-init = "./debian/cloud-init"
        ctx = "./debian/context"
    }
    args = {
        BASE_OS_NAME = "ubuntu"
        BASE_OS_VERSION = "24.04"
        DOCKER_APT_OS = "ubuntu"
        DOCKER_APT_CHANNEL ="noble"
        FRR_VERSION ="frr-10"
        FRR_VERSION_DETAIL ="10.4.1-0~ubuntu24.04.1"
        FRR_APT_CHANNEL ="noble"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
        # see https://kernel.ubuntu.com/mainline for available versions
        UBUNTU_MAINLINE_KERNEL_VERSION = "v6.12.44"
    }
    tags = ["ghcr.io/metal-stack/ubuntu:${SEMVER}"]
}

target "ubuntu-firewall" {
    inherits = ["_common"]
    dockerfile = "./firewall/Dockerfile"
    contexts = {
        baseapp = "target:ubuntu"
        ctx = "./firewall/context"
    }
    args = {
        BASE_OS_VERSION = "24.04"
        BASE_OS_NAME = "ghcr.io/metal-stack/ubuntu"
        SEMVER_MAJOR_MINOR = "3.0-ubuntu"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    tags = ["ghcr.io/metal-stack/firewall:3.0-ubuntu${SEMVER_PATCH}"]
}

