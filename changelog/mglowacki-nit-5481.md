### Deprecated

- The batch poster's dynamic price calculation and fallback from blobs to calldata — and the `--node.batch-poster.parent-chain-eip7623` flag that feeds it — is deprecated post-Amsterdam and may not function correctly if used. A warning is now logged when `--node.batch-poster.ignore-blob-price` is not set to `true`. Set `ignore-blob-price=true` to always post blobs when the parent chain supports them.
