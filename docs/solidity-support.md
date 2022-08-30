# Solidity Support

Arbitrum Nitro chains are Ethereum compatible, and therefore allow you to trustlessly deploy Solidity contracts (as well as Vyper or any other language that compiles to EVM bytecode).

# Differences from Solidity on Ethereum

Although Arbitrum supports Solidity code, there are differences in the effects of a few operations, including language features that don't make much sense in the Layer 2 context:

- `blockhash(x)` returns a cryptographically insecure, pseudo-random hash for `x` within the range `block.number - 256 <= x < block.number`. If `x` is outside of this range, `blockhash(x)` will return `0`. This includes `blockhash(block.number)`, which always returns `0` just like on Ethereum. The hashes returned do not come from L1.
- `block.coinbase` returns zero
- `block.difficulty` returns the constant 2500000000000000
- `block.number` / `block.timestamp` return an "estimate" of the L1 block number / timestamp at which the Sequencer received the transaction (see [Time in Arbitrum](./time.md))
- `msg.sender` works the same way it does on Ethereum for normal L2-to-L2 transactions; for L1-to-L2 "retryable ticket" transactions, it will return the L2 address alias of the L1 contract that triggered the message. See [retryable ticket address aliasing](./arbos/l1-to-l2-messaging.md#address-aliasing) for more.
