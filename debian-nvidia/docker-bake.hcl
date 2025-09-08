group "default" {
    targets = ["debian-nvidia"]
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

# declare empty var in order to be able to get value from shell
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}

target "debian-nvidia" {
    inherits = ["_common"]
    args = {
        BASE_OS_VERSION = 12
        BASE_OS_NAME = "ghcr.io/metal-stack/debian"
      # see https://packages.debian.org/bookworm/kernel/ for available versions
        KERNEL_VERSION = "6.1.0-38"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/debian:12"]
    context = "."
}
