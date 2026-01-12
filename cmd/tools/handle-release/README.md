# handle-release

## Run locally

```bash
FILENAME=downloads.md \
REF_NAME=$(git rev-parse --abbrev-ref HEAD) \
DISTRO_VERSIONS=$'["capms-ubuntu/1.32.9", "ubuntu/24.04"]' \
go run ./cmd/tools/handle-release --dry-run
```
