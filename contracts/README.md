# Contracts

This folder contains the smart contracts needed for the rollup protocol, which are
a solidity implementation of the specification meant to match the behavior of the Go
implementation also contained in this repository.

This subfolder was initialized using [Foundry](https://github.com/foundry-rs/foundry) with `forge init`

## Setup

Requirements: [nvm](https://github.com/nvm-sh/nvm)

```sh
# Use nvm to install node 16.x
nvm install 16
nvm use 16

# Install yarn, if you don't have it already
npm i -g yarn

# Run yarn to install deps
yarn

# Install forge deps
forge install
```

## Run Tests

```
forge test
```
