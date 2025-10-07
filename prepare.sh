#!/bin/bash
set -e

OS_FLAVOR=$1

echo "Setting GitHub environment variables"

BRANCH=$(echo "${GITHUB_REF}" | awk -F / '{print $3}')
echo "::set-env name=BRANCH::${BRANCH})"

if [ -n "${GITHUB_BASE_REF}" ]; then
    # this is a pull request build
    PULL_REQUEST_NUMBER=$(echo "$GITHUB_REF" | awk -F / '{print $3}')
    BRANCH="${GITHUB_HEAD_REF##*/}"
    echo "::set-env name=OUTPUT_FOLDER::/${PULL_REQUEST_NUMBER}-${BRANCH}"
else
    if [ "$BRANCH" = "master" ]; then
        # this is a build from stable branch
        echo "::set-env name=OUTPUT_FOLDER::/stable"
    else
        # this is a release build
        echo "::set-env name=SEMVER_PATCH::$(date +%Y%m%d)"
        echo "::set-env name=OUTPUT_FOLDER::/"
    fi
fi

echo "Generating build metadata"
mkdir -p "${OS_FLAVOR}/context/etc/metal"
BUILD_META_FILE="${OS_FLAVOR}/context/etc/metal/build.yaml"
python3 - <<EOF | tee -a "${BUILD_META_FILE}"
from datetime import datetime
print(f"builddate='{datetime.now().isoformat(' ')}'")
print("commit_ref='${BRANCH}'")
print("commit_sha1='${GITHUB_SHA}'")
print("gitrepo='${GITHUB_REPOSITORY}'")
EOF

echo "remove old firecracker images"
sudo rm -rf /var/lib/firecracker/image/* /var/lib/firecracker/kernel/*

echo "create tarball output directory"
mkdir -p "images/${PULL_REQUEST_NUMBER}-${BRANCH}/${OS_FLAVOR}/${SEMVER_MAJOR_MINOR}"
