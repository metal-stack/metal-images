# Image store

The metal-images tarballs are stored on <https://images.metal-stack.io/>. The tarball must be compressed using lz4 and a md5 checksum must be provided as well. To be able to have an insight what packages are included in this image a `packages.txt` with the output of `dpkg -l`.
The actual directory layout should look like:

```bash
<imagesdir>/<os>/<major.minor>/<patch>/img.tar.lz4
<imagesdir>/<os>/<major.minor>/<patch>/img.tar.lz4.md5
<imagesdir>/<os>/<major.minor>/<patch>/packages.txt
```

Where `<imagesdir>` is `/` for the master branch and `/${CI_COMMIT_REF_SLUG}/` for branches and merge requests.

`<os>` is the name of the os in use, some images like `firewall` are derived from another os image (in this case the `ubuntu` image).

`<major.minor>` specifies the major and minor number of the OS, which is case of ubuntu "19.10", "19.10", "20.04" and so on. This version must follow the semantic versioning specification, whereas we tolerate a leading zero for the minor version which is quite common for some OSes.

`<patch>` must follow the semantic version requirements for `patch`, we defined that patch is always in the form of "YYYYMMDD` for example 20191018.

To specify the image for machine creation the full qualified image must be in the form of:
`<os-major.minor.patch>`, e.g. `ubuntu-19.10.20191018`.

From the metal-api perspective, there are two possibilities to specify a image to create a machine:

1. specify major.minor without patch, e.g. `--image ubuntu-19.10`
1. specify major.minor.patch `--image ubuntu-19.10.20191018`

In the first case a most recent version resolution is taken place in the metal-api to resolve to the most recent available image for ubuntu-19.10, which will be then for example ubuntu-19.10.20191018, this image is the stored in the machine allocation.
The second form guarantees the machine creation of this exact image.

Images which are no longer in use by any allocated machine and are older than the specified usage period will be deleted from the metal-api and the image store.
