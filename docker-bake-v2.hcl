target "_common" {
    attest = ["type=sbom"]
    output = [
        "type=tar,dest=./img.tar"
    ]
}

target "_common_args" {
    args = {
        IGNITION_BRANCH = "v0.36.2"
        GOLLDPD_VERSION = "v0.4.12"
        CRI_VERSION = "v1.36.0"
        ICE_VERSION = "2.4.5"
        ICE_PKG_VERSION = "1.3.53.0"
    }
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
        BASE_OS_VERSION = "trixie"
        DOCKER_APT_OS = "debian"
        DOCKER_APT_CHANNEL ="trixie"
        FRR_VERSION ="frr-10.7"
        FRR_VERSION_DETAIL ="10.7.1-0~deb13u1"
        FRR_APT_CHANNEL ="trixie"
        # see https://packages.debian.org/trixie/kernel/ for available versions
        KERNEL_VERSION = "6.12.111+deb13"
        CONTAINERD_VERSION = "2.2.6-1~debian.13~trixie"
    }
}
