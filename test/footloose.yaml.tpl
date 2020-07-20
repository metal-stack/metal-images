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
      kernel: weaveworks/ignite-kernel:5.4.43
      cpus: 2
      memory: 1GB
      diskSize: 4GB
