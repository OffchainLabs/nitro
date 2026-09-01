### Internal
- nitro-reth links the prebuilt brotli static libs (`scripts/build-brotli.sh -l`) like the rest of the repo instead of compiling brotli's C sources per-build via `cc_brotli`
