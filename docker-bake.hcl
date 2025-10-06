# declare empty var in order to be able to get value from shell
variable "OS_NAME" {}
variable "OUTPUT_FOLDER" {}
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}

target "_common" {
    attest = [
        "type=provenance,mode=max",
        "type=sbom",
    ]
    output = [
        "type=registry",
        "type=tar,dest=./images${OUTPUT_FOLDER}/${OS_NAME}/${SEMVER_MAJOR_MINOR}/img.tar"
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
    tags = ["ghcr.io/metal-stack/almalinux:${SEMVER_MAJOR_MINOR}${SEMVER_PATCH}"]
}

target "debian" {
    inherits = ["_common", "_common_args"]
    dockerfile = "./debian/Dockerfile"
    contexts = {
        cloudinit = "./debian/cloud-init"
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
        KERNEL_VERSION = "6.1.0-40"
    }
    tags = ["ghcr.io/metal-stack/debian:${SEMVER_MAJOR_MINOR}${SEMVER_PATCH}"]
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
    tags = ["ghcr.io/metal-stack/debian-nvidia:${SEMVER_MAJOR_MINOR}${SEMVER_PATCH}"]
}

target "ubuntu" {
    inherits = ["_common", "_common_args"]
    dockerfile = "./debian/Dockerfile"
    contexts = {
        cloudinit = "./debian/cloud-init"
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
        UBUNTU_MAINLINE_KERNEL_VERSION = "v6.12.54"
    }
    tags = ["ghcr.io/metal-stack/ubuntu:${SEMVER_MAJOR_MINOR}${SEMVER_PATCH}"]
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

variable "KUBE_VERSION" {}
variable "KUBE_APT_BRANCH" {}

target "capms" {
    inherits = ["_common"]
    dockerfile = "./capms/Dockerfile"
    contexts = {
        baseapp = "target:ubuntu"
        ctx = "./capms/context"
    }
    args = {
        BASE_OS_NAME    = "ghcr.io/metal-stack/ubuntu"
        BASE_OS_VERSION = "24.04"

        KUBE_APT_BRANCH  = "${KUBE_APT_BRANCH}"
        KUBE_VERSION = "${KUBE_VERSION}"
        KUBE_VIP_VERSION = "v0.8.10"

        CRANE_CHECKSUM = "sha256:36c67a932f489b3f2724b64af90b599a8ef2aa7b004872597373c0ad694dc059"
        CRANE_RELEASE  = "https://github.com/google/go-containerregistry/releases/download/v0.20.3/go-containerregistry_Linux_x86_64.tar.gz"
    }
    tags = ["ghcr.io/metal-stack/capms:${SEMVER_MAJOR_MINOR}-ubuntu-24.04${SEMVER_PATCH}"]
}
