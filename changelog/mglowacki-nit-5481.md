### Deprecated

- The batch poster's dynamic price calculation and fallback from blobs to calldata — and the `--node.batch-poster.parent-chain-eip7623` flag that feeds it — is deprecated post-Amsterdam and may not function correctly if used. A warning is now logged when `--node.batch-poster.ignore-blob-price` is set to `false`. When blob posting is enabled (`--node.batch-poster.post-4844-blobs`) and the parent chain supports blobs, set `--node.batch-poster.ignore-blob-price=true` to always post blobs (disabling the deprecated dynamic price comparison).

### Configuration

- `--node.batch-poster.ignore-blob-price` now defaults to `true`: when blob posting is enabled (`--node.batch-poster.post-4844-blobs`) the batch poster posts blobs unconditionally, without the deprecated dynamic price comparison. Set `ignore-blob-price=false` to restore the deprecated dynamic price fallback.
