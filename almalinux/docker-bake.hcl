group "default" {
    targets = ["almalinux"]
}

target "_common" {
    args = {
        IGNITION_BRANCH = "v0.36.2"
        GOLLDPD_VERSION = "v0.4.9"
    }
}

# declare empty var in order to be able to get value from shell
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}

target "almalinux" {
    inherits = ["_common"]
    args = {
        BASE_OS_VERSION = 9
        FRR_VERSION="frr-stable"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/almalinux:9"]
    context = "."
}
