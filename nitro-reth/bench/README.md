# bench/

Performance corpus for `arb-bench`. See `crates/arb-bench/README.md` for usage.

## Layout

- `corpus/synthetic/` — Generated in-process by `arbreth-bench`. Self-contained; no
  network access required. These are the manifests that the local + PR-gate flows
  use.
- `corpus/sepolia/` — Manifests that point to frozen Sepolia tarballs published as
  GitHub Release assets (`bench-corpus-vX.Y.Z`). Use `arbreth-bench capture` +
  `arbreth-bench curate` to refresh these.
- `corpus/configs/` — Node config variants exercised by the matrix.
- `baselines/master/` — Recorded run results per master commit. Used for trend
  tracking and as the `compare` baseline. Auto-committed by `bench-nightly.yml`.
- `baselines/CORPUS_VERSION.txt` — Pin the corpus version; bumping invalidates
  historical comparisons.

## Refreshing the frozen corpus (quarterly)

1. `arbreth-bench capture --rpc <ARB_SEPOLIA_RPC> --from N --to M --out raw.json`
2. `arbreth-bench curate --input raw.json --out staging --corpus-version 2.0.0`
3. `tar -I 'zstd -19' -cf bench-corpus-v2.0.0.tar.zst staging`
4. Publish as a GitHub Release; bump `messages.sha256` in each `sepolia/*` manifest.
5. Bump `CORPUS_VERSION.txt`; note the version change in PR description.
