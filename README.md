# Challenge Protocol V2

[![Go](https://github.com/OffchainLabs/challenge-protocol-v2/actions/workflows/go.yml/badge.svg)](https://github.com/OffchainLabs/challenge-protocol-v2/actions/workflows/go.yml)

This repository implements V2 of the Arbitrum challenge protocol. V2 is an efficient, all-vs-all dispute protocol that enables anyone on Ethereum L1 to challenge incorrect Arbitrum state transitions in a permissionless manner. Given Arbitrum’s state transition is deterministic, this guarantees only one correct result at any given assertion. A **single, honest participant** will always win against malicious entities when challenging assertions posted to Ethereum. 

The code in this repository will eventually be migrated to [github.com/offchainlabs/nitro](https://github.com/offchainlabs/nitro), which includes the necessary execution machines required for interacting iwth the protocol.

## Dependencies

- [Go v1.19](https://go.dev/doc/install)
- Bazelisk tool to install the Bazel build system

Bazelisk can be installed globally using the Go tool:

```
go install github.com/bazelbuild/bazelisk@latest
```

Then, we recommend aliasing the `bazel` command to `bazelisk`

## Building the Go Code

The project can be built with either the Go tool or the Bazel build system. We use [Bazel](https://bazel.build) because it provides a hermetic, deterministic environment for building our project and gives us access to many tools including a suite of **static analysis checks**.

Once bazelisk in

## Running Go Tests

## Regenerating Solidity Bindings

* Install nodejs and npm
* Install yarn with `npm i -g yarn`
* In the `contracts/` directory, run `yarn install` then `yarn --cwd contracts build`
* In the **top-level directory**, run `go run ./solgen/main.go`
* You should now have Go bindings inside of `solgen/go`

## Running Solidity Tests
