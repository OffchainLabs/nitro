# Gas Dimension Test Contracts

This folder contains smart contracts that force various types of opcodes to run in different configurations. This is used to test gas accounting across multiple gas dimensions, e.g. computation, state read/write, storage growth, etc.

# Scripts for Debugging

The scripts for running the contracts are intended to be used with forge script, e.g.

```bash
forge script gas-dimensions/scripts/Sstore.s.sol -vvvvv --private-key "nitro.dev.node.private.key.here" --slow --broadcast --rpc-url http://127.0.0.1:8547 --priority-gas-price "1000000000" --with-gas-price "2000000000" -g "10000"  --chain-id "412346"
```

These scripts can be helpful when manually debugging the code in `gas-dimensions/src/` against a local dev node.
