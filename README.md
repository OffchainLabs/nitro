# Challenge Protocol V2

[![Go](https://github.com/OffchainLabs/challenge-protocol-v2/actions/workflows/go.yml/badge.svg)](https://github.com/OffchainLabs/challenge-protocol-v2/actions/workflows/go.yml)

This repository implements V2 of the Arbitrum challenge protocol. V2 is an efficient, all-vs-all dispute protocol that enables anyone on Ethereum L1 to challenge incorrect Arbitrum state transitions in a permissionless manner. Given Arbitrum’s state transition is deterministic, this guarantees only one correct result at any given assertion. A **single, honest participant** will always win against malicious entities when challenging assertions posted to Ethereum. 

The code in this repository will eventually be migrated to [github.com/offchainlabs/nitro](https://github.com/offchainlabs/nitro), which includes the necessary execution machines required for interacting with the protocol.

A complete list of reference documentation for the repository can be found on Notion [here](https://www.notion.so/arbitrum/Challenge-Protocol-V2-Trail-of-Bits-Kickoff-cf3b54ba0b234b0195bfdd08c6cbcc88)

## Dependencies

- [Go v1.19](https://go.dev/doc/install)
- Bazelisk tool to install the Bazel build system
- Node.js v14, we recommend using [node version manager](https://github.com/nvm-sh/nvm)

Bazelisk can be installed globally using the Go tool:

```
go install github.com/bazelbuild/bazelisk@latest
```

Then, we recommend aliasing the `bazel` command to `bazelisk`

```
alias bazel=bazelisk
```


## Building the Go Code

```
git clone git@github.com:offchainlabs/challenge-protocol-v2 && cd challenge-protocol-v2
```

The project can be built with either the Go tool or the Bazel build system. We use [Bazel](https://bazel.build) because it provides a hermetic, deterministic environment for building our project and gives us access to many tools including a suite of **static analysis checks**, and a great dependency management approach.

To build, simply do:

```
bazel build //...
```

To build a specific target, do

```
bazel build //util/prefix-proofs:go_default_library
```

More documentation on common Bazel commands can be found [here](https://bazel.build/reference/command-line-reference)

The project can also be ordinarily built with the Go tool

``` 
go build ./...
```

## Running Go Tests

Running tests with Bazel can be done as follows:

```
bazel test //...
```

To run a specific target, do:

```
bazel test //util/prefix-proofs:go_default_library
```

To see outputs, run the test multiple times, or pass in specific arguments to the Go test:

```
bazel test //util/prefix-proofs:go_default_test --runs_per_test=10 --test_filter=<TEST_NAME_HERE> --test_output=streamed
```

Tests can also be run ordinarily with the Go tool

```
go test ./...
```

### Fuzz Tests

The repo contains a few fuzz tests using Go's fuzzer, within the `util/` package. To run an example one, do:

```
go test -fuzz=FuzzVerify -fuzztime=10m -run=FuzzVerify ./util/prefix-proofs
```

## Regenerating Solidity Bindings

With node version 14 and npm, install `yarn`

```
npm i -g yarn
```

Then install Node dependencies

```
cd contracts && yarn install
```

Building the contracts can be done with:

```
yarn --cwd contracts build
```

To generate the Go bindings to the contracts, at the **top-level directory**, run:

```
go run ./solgen/main.go
```

You should now have Go bindings inside of `solgen/go`

## Running Solidity Tests

Solidity tests can be run using hardhat, but we recommend using Foundry as the tool of choice

```
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

In the contracts folder, run

```
forge test
```

Output:

```
Test result: ok. 42 passed; 0 failed; finished in 1.60s
```
