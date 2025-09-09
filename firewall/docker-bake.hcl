group "default" {
    targets = ["debian", "ubuntu"]
}

# declare empty var in order to be able to get value from shell
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}
variable "SEMVER" {}

target "debian" {
    args = {
        BASE_OS_VERSION = 12
        BASE_OS_NAME = "ghcr.io/metal-stack/debian"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/firewall:${SEMVER}"]
    context = "."
}

target "ubuntu" {
    args = {
        BASE_OS_VERSION = "24.04"
        BASE_OS_NAME = "ghcr.io/metal-stack/ubuntu"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/firewall:${SEMVER}"]
    context = "."
}
