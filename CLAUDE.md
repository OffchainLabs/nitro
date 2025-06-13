# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is Arbitrum Nitro with EigenDA integration - a Layer 2 Ethereum rollup solution.

## Key Directories

- `arbnode/`: Core L2 node implementation
- `arbos/`: Layer 2 operating system 
- `arbitrator/`: WASM-based fraud proof system
- `system_tests/`: End-to-end integration tests
- `eigenda/`: EigenDA client and integration logic
- `contracts/`: Solidity smart contracts
- `bold/`: BOLD challenge protocol implementation

## Development Commands

### Building
```bash
make                          # Development workflow: lint + test + format check
make all                      # Build everything: all binaries + replay env + test proofs
make build                    # Build all binaries only
make build-node-deps          # Build dependencies for node
make build-prover-lib         # Build arbitrator/prover library
make build-replay-env         # Build replay environment
```

### Testing
```bash
make test-go                  # Run Go tests
make test-rust                # Run Rust arbitrator tests
make tests                    # Run both Go and Rust tests
make tests-all                # Run all tests including slow/unreliable ones

# Specific test categories
make test-go-challenge        # Run challenge/fraud proof tests
make test-go-stylus          # Run Stylus WASM execution tests  
make test-go-redis           # Run Redis-dependent tests

# EigenDA integration tests
# MUST start proxy first with this shell script. If it fails, that likely means that the proxy has already been started
./scripts/start-eigenda-proxy.sh
go test -timeout 600s -run ^TestEigenDAIntegration$ github.com/offchainlabs/nitro/system_tests
```

### Linting and Formatting
```bash
make lint                     # Run Go linters and basic checks
make fmt                      # Format Go and Rust code
```

### Development Utilities
```bash
make clean                    # Clean build artifacts and test cache
make wasm-ci-build           # Build WASM components for CI
make stylus-benchmarks       # Run Stylus performance benchmarks
```