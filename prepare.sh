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
python3 -c "import yaml; from datetime import datetime; print(yaml.dump(dict(builddate=datetime.now(), commit_ref=\"${BRANCH}\", commit_sha1=\"${GITHUB_SHA}\", gitrepo=\"${GITHUB_REPOSITORY}\"), default_flow_style=False))" | tee -a "${BUILD_META_FILE}"

echo "remove old firecracker images"
sudo rm -rf /var/lib/firecracker/image/* /var/lib/firecracker/kernel/*
