ARG BASE_OS_NAME
ARG BASE_OS_VERSION

FROM ${BASE_OS_NAME}:${BASE_OS_VERSION}

ARG KUBE_APT_BRANCH
ARG KUBE_VERSION

ENV DEBCONF_NONINTERACTIVE_SEEN="true" \
    DEBIAN_FRONTEND="noninteractive"

COPY context/ /

RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/${KUBE_APT_BRANCH}/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/${KUBE_APT_BRANCH}/deb/ /" > /etc/apt/sources.list.d/kubernetes.list \
 && apt update \
 && apt install --yes --no-install-recommends \
        kubeadm=${KUBE_VERSION}-1.1 \
        kubectl=${KUBE_VERSION}-1.1 \
        kubelet=${KUBE_VERSION}-1.1 \
        kubernetes-cni
