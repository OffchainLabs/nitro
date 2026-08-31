# AGENTS.md

Guidance for coding agents working in this repository.

## Common pitfalls

- Building the Go code requires the contract Go bindings. If the build fails with missing packages under `solgen`, run `make contracts` first.
- Linking native nitro-reth binaries (tests included) requires the prebuilt brotli static libs. If the build fails with `could not find native static library brotlienc-static`, run `./scripts/build-brotli.sh -l` (needs cmake) from the repo root first.
- Running system tests (`system_tests/`) requires building test dependencies first. Run `make test-go-deps` before running them.
- Every PR requires a changelog file in the `changelog/` dir (e.g. `changelog/<author>-<issue>.md`) with the format:

  ```
  ### <Section>
  - <Short description of the change>
  ```

  The available sections are listed in `changelog/.unclog.yaml`.
