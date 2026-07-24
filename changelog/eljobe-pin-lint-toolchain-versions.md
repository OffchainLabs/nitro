### Internal
- Pinned the Go toolchain and golangci-lint versions used by `make` to the same versions CI uses, and switched the CI lint job to run `make lint` so the pins live in one place (the Makefile). `make lint` no longer applies automatic fixes; use the new `make lint-fix` target for that.
