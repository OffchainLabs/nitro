# Copyright 2026, Offchain Labs, Inc.
# For license information, see https://github.com/OffchainLabs/arbitrum-reth/blob/main/LICENSE.md

##@ Help

.PHONY: help ## Display this help.
help:
	@awk 'BEGIN { printf "Usage:\n  make \033[36m<target>\033[0m\n" } /^##@ / { section = substr($$0, 5); printed = 0; next } /^\.PHONY:/ { desc = ""; line = $$0; if (index(line, "##")) { desc = line; sub(/^.*##[[:space:]]*/, "", desc); sub(/[[:space:]]+$$/, "", desc); sub(/[[:space:]]*##.*$$/, "", line) } sub(/^\.PHONY:[[:space:]]*/, "", line); if (!printed && section != "") { printf "\n\033[1m%s\033[0m\n", section; printed = 1 } targets_len = split(line, targets, /[[:space:]]+/); for (i = 1; i <= targets_len; i++) if (targets[i] != "") printf "  \033[36m%-28s\033[0m %s\n", targets[i], desc }' $(MAKEFILE_LIST)

##@ Test

.PHONY: test ## Run all tests.
test:
	cargo test --workspace -- --nocapture

##@ Other

.PHONY: fmt ## Run `rustfmt` on the entire workspace.
fmt: 
	cargo +nightly fmt --all

.PHONY: clippy ## Run `clippy` on the entire workspace.
clippy: 
	cargo clippy --all --all-targets --no-deps -- --deny warnings

.PHONY: lint ## Run all linters.
lint: fmt clippy

.PHONY: check ## Run `cargo check`.
check:
	cargo check --workspace

.PHONY: clean ## Run `cargo clean`.
clean:
	cargo clean

.PHONY: pr ## Run all checks for a PR.
pr: lint check test
