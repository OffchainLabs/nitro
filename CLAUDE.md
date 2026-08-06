# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Common pitfalls

- Building the Go code requires the contract Go bindings. If the build fails with missing packages under `solgen`, run `make contracts` first.
- Running system tests (`system_tests/`) requires building test dependencies first. Run `make test-go-deps` before running them.
