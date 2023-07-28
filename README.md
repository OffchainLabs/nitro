# BOLD

[![Go Report Card](https://goreportcard.com/badge/github.com/OffchainLabs/bold)](https://goreportcard.com/report/github.com/OffchainLabs/bold)
[![codecov](https://codecov.io/gh/OffchainLabs/bold/branch/main/graph/badge.svg)](https://codecov.io/gh/OffchainLabs/bold)

[![Go](https://github.com/OffchainLabs/bold/actions/workflows/go.yml/badge.svg)](https://github.com/OffchainLabs/bold/actions/workflows/go.yml)

This repository implements Offchain Labs' BOLD (Bounded Liquidity Delay) Protocol: a dispute system to enable permissionless validation of Arbitrum chains. It is an efficient, all-vs-all challenge protocol that enables anyone on Ethereum to challenge invalid rollup state transitions. Given state transitions are deterministic, this guarantees only one correct result for any given assertion. A **single, honest participant** will always win against malicious entities when challenging assertions posted to the settlement chain. 

## Repository Structure

For detailed information on how our code is architected and how it meets the BOLD specification, see [ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
api/ 
    API for monitoring and visualizing challenges
assertions/
    Logic for scanning and posting assertions
chain-abstraction/
    High-level wrappers around Solidity bindings for the Rollup contracts
challenge-manager/
    All logic related to challenging, managing challenges
containers/
    Data structures used in the repository, including FSMs
contracts/
    All Rollup / challenge smart contracts
docs/
    Diagrams and architecture
layer2-state-provider/
    Interface to request state and proofs from an L2 backend
math/
    Utilities for challenge calculations
runtime/
    Tools for managing function lifecycles
state-commitments/
    Proofs, history commitments, and Merkleizatins
testing/
    All non-production code
third_party/
    Build artifacts for dependencies
time/
    Abstract time utilities
```

## Building

### Go Code

Install [Go v1.19](https://go.dev/doc/install). Then:

```
git clone https://github.com/OffchainLabs/bold.git && cd bold
```

The project can be built with either the Go tool or the Bazel build system. We use [Bazel](https://bazel.build) internally because it provides a hermetic, deterministic environment for building our project and gives us access to many tools including a suite of **static analysis checks**, and a great dependency management approach.

##### With Go

To build, simply do:

``` 
go build ./...
```

##### With Bazel

We recommend getting the [Bazelisk](https://github.com/bazelbuild/bazelisk) tool to install the Bazel build system. Bazelisk can be installed globally using the Go tool:

```
go install github.com/bazelbuild/bazelisk@latest
```

Then, we recommend aliasing the `bazel` command to `bazelisk`

```
alias bazel=bazelisk
```

To build with Bazel, 
```
bazel build //...
```

To build a specific target, do

```
bazel build //util/prefix-proofs:go_default_library
```

More documentation on common Bazel commands can be found [here](https://bazel.build/reference/command-line-reference)

The project can also be ordinarily built with the Go tool

## Testing

### Running Go Tests

Install [Foundry](https://book.getfoundry.sh/getting-started/installation) to get the `anvil` command locally, which allows setting up a local Ethereum chain for testing purposes. Next:

```
go test ./...
```

Alternatively, tests can be ran with Bazel as follows:

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

### Running Solidity Tests

Solidity tests can be run using hardhat, but we recommend using [Foundry](https://book.getfoundry.sh/getting-started/installation) as the tool of choice

In the contracts folder, run:

```
forge test
```

Output:

```
Test result: ok. 42 passed; 0 failed; finished in 1.60s
```

## Generating Solidity Bindings

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

## License

BOLD uses [Business Source License 1.1](./LICENSE)