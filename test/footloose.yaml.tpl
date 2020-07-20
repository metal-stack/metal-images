---
cluster:
  name: imagevm
  privateKey: key
machines:
- count: 1
  spec:
    backend: ignite
    image: ${IMAGE}
    name: ${OS_NAME}%d
  ignite:
    runtime: docker
    cpus: 2
    memory: 1GB
    diskSize: 4GB
    kernel: weaveworks/ignite-kernel:4.19.47
