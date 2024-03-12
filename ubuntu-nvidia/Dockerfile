ARG BASE_OS_NAME
ARG BASE_OS_VERSION

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive" \
    SEMVER_MAJOR_MINOR=${SEMVER_MAJOR_MINOR}

RUN set -ex \
 && rm -f /.dockerenv \
 && apt-get remove --yes linux-image* linux-modules* \
 && apt-get update \
 && apt-get upgrade --yes \
 && apt-get install --yes \
    linux-generic

