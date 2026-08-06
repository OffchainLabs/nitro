# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Common pitfalls

- Building the Go code requires the contract Go bindings. If the build fails with missing packages under `solgen`, run `make contracts` first.
- Running system tests (`system_tests/`) requires building test dependencies first. Run `make test-go-deps` before running them.
- When submitting a new feature or fix, add a changelog file to the `changelog/` dir (e.g. `changelog/<author>-<issue>.md`) with the format:

  ```
  ### <Section>
  - <Short description of the change>
  ```

  The available sections are listed in `changelog/.unclog.yaml`.
