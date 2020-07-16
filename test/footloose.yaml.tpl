---
cluster:
  name: cluster
  privateKey: key
machines:
- count: 1
  spec:
    backend: ignite
    image: ${IMAGE}
    name: images%d
  ignite:
    runtime: docker
    cpus: 2
    memory: 1GB
    diskSize: 4GB
    kernel: weaveworks/ignite-kernel:4.19.47
