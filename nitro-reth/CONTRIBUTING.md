# Contributing to Arbitrum Reth

Thanks for your interest in contributing to Arbitrum Reth.

## Getting Started

```bash
git clone https://github.com/0xBloctopus/arbitrum-reth.git
cd arbitrum-reth
cargo check
cargo test
```

**Requirements:** Rust 1.93+, clang, cmake

## Development Workflow

1. Fork the repository and create a feature branch
2. Make your changes
3. Ensure all checks pass:
   ```bash
   cargo +nightly fmt --all
   cargo clippy --workspace --all-targets -- -D warnings
   cargo test --workspace
   cargo doc --workspace --no-deps
   ```
4. Open a pull request against `master`

## Code Standards

- Use `alloy-primitives` types (`Address`, `B256`, `U256`) throughout
- Use `thiserror` for error types
- Use `#[derive(Debug, Clone)]` on structs
- No `unwrap()` in library code — use `?` and proper error types
- Document public APIs with doc comments
- Follow existing reth patterns for crate and module structure

## Pull Requests

- Open an issue before starting work on larger changes
- Keep PRs focused — one logical change per PR
- Include a clear description of what changed and why
- Ensure CI passes before requesting review

## Reporting Issues

- **Bugs:** Open a GitHub issue with reproduction steps
- **Security vulnerabilities:** See [SECURITY.md](SECURITY.md) — do not file a public issue

## License

By contributing, you agree that your contributions will be licensed under the [Business Source License 1.1](LICENSE.md).
