#!/bin/bash
set -e

OS_FLAVOR=$1

echo "Setting GitHub environment variables"

ref="${GITHUB_REF}"
if [ ! -z "${GITHUB_BASE_REF}" ]; then
    echo "using GITHUB_BASE_REF to determine branch name"
    ref="${GITHUB_BASE_REF}"
fi

BRANCH=$(echo "${ref}" | awk -F / '{print $3}')
echo "::set-env name=BRANCH::${BRANCH})"

if [ "${BRANCH}" != "master" ]; then
    echo "::set-env name=OUTPUT_FOLDER::/${BRANCH}"
fi
echo "::set-env name=SEMVER_PATCH::$(date +%Y%m%d)"

echo "Generating build metadata"
mkdir -p ${OS_FLAVOR}/context/etc/metal
BUILD_META_FILE="${OS_FLAVOR}/context/etc/metal/build.yaml"
python -c "import yaml; from datetime import datetime; print yaml.dump(dict(builddate=datetime.now(), commit_ref=\"${BRANCH}\", commit_sha1=\"${GITHUB_SHA}\", gitrepo=\"${GITHUB_REPOSITORY}\"), default_flow_style=False)" | tee -a ${BUILD_META_FILE}