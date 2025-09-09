group "default" {
    targets = ["debian-nvidia"]
}

# declare empty var in order to be able to get value from shell
variable "SEMVER_PATCH" {}
variable "SEMVER_MAJOR_MINOR" {}

target "debian-nvidia" {
    args = {
        BASE_OS_VERSION = 12
        BASE_OS_NAME = "ghcr.io/metal-stack/debian"
        SEMVER_MAJOR_MINOR = "${SEMVER_MAJOR_MINOR}"
        SEMVER_PATCH = "${SEMVER_PATCH}"
    }
    dockerfile = "Dockerfile"
    tags = ["ghcr.io/metal-stack/debian-nvidia:12"]
    context = "."
}
