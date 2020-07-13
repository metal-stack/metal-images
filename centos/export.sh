#!/usr/bin/env bash
set -ex

readonly SEMVER=${SEMVER_MAJOR_MINOR}${SEMVER_PATCH}
readonly DOCKER_IMAGE="quay.io/metalstack/${OS_NAME}:${SEMVER}"
readonly IMAGE_BASENAME=img

readonly SEMVER_PATCH_DIR=$(echo ${SEMVER_PATCH} | tr -d ".")
readonly TARGET_PATH="images${OUTPUT_FOLDER}/${OS_NAME}/${SEMVER_MAJOR_MINOR}/${SEMVER_PATCH_DIR}"
readonly EXPORT_DIRECTORY="../${TARGET_PATH}"

readonly TAR="${IMAGE_BASENAME}.tar"
readonly LZ4="${IMAGE_BASENAME}.tar.lz4"
readonly MD5="${IMAGE_BASENAME}.tar.lz4.md5"
readonly PKG="packages.txt"

# export tarball with checksum from docker image and package list
mkdir -p ${EXPORT_DIRECTORY}
cd ${EXPORT_DIRECTORY}
docker export $(docker create ${DOCKER_IMAGE}) > ${TAR}
# FIXME unify with global export
docker run --rm ${DOCKER_IMAGE} bash -c "rpm -qa" > ${PKG}
lz4 -f -9 ${TAR} ${LZ4}
rm -f ${TAR}
md5sum ${LZ4} > ${MD5}

# export a list with the generated fqdn image names
# mkdir -p workdir
echo "${OS_NAME}-${SEMVER_MAJOR_MINOR}-${SEMVER_PATCH}" 
