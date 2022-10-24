# Nitro Migration Notes for Solidity Devs: Living Document

The following is a summary of changes in the Arbitrum One chain's 8/31/22 Nitro upgrade that dapp devs should be aware of.

_Last Updated 31 Aug 2022_

## Cool New Stuff

For starters, here's a sampling of exciting perks dapps get with the Nitro upgrade:

- **Ethereum L1 Gas Compatibility 🥳**: Gas pricing and accounting for EVM operations is be perfectly in line with L1; no more ArbGas.
    - Although in Nitro L2 opcodes are now priced the exact same as Eth L1 - you still have to pay for the L1 calldata (which gets represented as L2 gas units). Your transactions' gas used won't be the same as Eth L1 but during contract execution the opcode cost and `gasleft` interactions are now better aligned with Eth L1. Read more [here](https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9).

- **Safer Retryable tickets 🥳**: Retryable tickets' submission cost is collected in the L1 Inbox contract; if the submission cost is too low, the transaction will simply revert on the L1 side, eliminating the [failure mode](https://developer.offchainlabs.com/docs/l1_l2_messages#important-note-about-base-submission-fee) in which a retryable ticket fails to get created.

- **Calldata compression 🥳**: Compression takes place protocol level; dapps don't need to change anything, data will just get cheaper! (You are charged even less if your calldata is highly compressible with brotli.)

- **Support for All Ethereum L1 precompiles 🥳**: (`blake2f`, `ripemd160`, etc)

- **Tighter Synchronization with L1 Block Numbers 🥳**: L1 block numbers (accessed via `block.number` on L2) are updated more frequently in Nitro than in Arbitrum classic; expect them to be nearly real-time/ in sync with L1.

- **Frequent Timestamps 🥳**: Timestamps (accessed via `block.timestamp` on L2) are updated every block based on the sequencer’s clock; i.e., it is no longer linked to the timestamp of the last L1 block.

- **L2 Block hash EVM Consistency 🥳**: L2 block hashes take the same format as on Ethereum (if you query it from the ArbSys precompile, not the one in `block.hash(uint256)`).

- **Geth tracing 🥳**: `debug_traceTransaction` RPC endpoint is supported; this includes tracing of ArbOS internal bookkeeping actions.

## Breaking changes

#### Dapps

- **Gas Accounting**: It's now consistent with the L1 EVM; L2 gas usage will change due to different accounting from ArbGas. Any hard-coded gas values should be changed accordingly (the same applies to any gas amount used in conjunction with `gasleft`). That said, you shouldn't be hard-coding any gas values anyway, just like you shouldn't in L1 Ethereum, since both the L1 and L2 gas schedule may change in the future.
    - Although in Nitro L2 opcodes are now priced the exact same as Eth L1 - you still have to pay for the L1 calldata (which gets represented as L2 gas units). Your transactions' gas used won't be the same as Eth L1 but during contract execution the opcode cost and `gasleft` interactions are now better aligned with Eth L1. Read more [here](https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9).

- **No more storage gas**: there is no more concept of a separate pool of storage gas, and opcodes are priced identically to the L1 EVM.

- **New L2 to L1 event signature**: The function signature for the [L2 to L1 event](../../contracts/src/precompiles/ArbSys.sol#L110) emitted by ArbSys has now changed.

- **Lower contract code size limit**: Contracts of up to 48KB were deployable, but now only up to 24KB are deployable (as specified in [EIP 170](https://eips.ethereum.org/EIPS/eip-170)). Previously deployed contracts above the limit will be maintained (but contracts deployed by these legacy contracts are capped by the new size).

- **Retryable Tickets**:
  - The submission cost is now enforced in the L1 inbox and checked against the L1 transaction's `msg.value`; contracts shouldn't rely on funds pooled in the L2 destination to cover this cost.
  - The current submission price is now not available in the L2 ArbRetryableTx precompile, instead it can be queried in the L1 Delayed Inbox [`calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee)`](https://github.com/OffchainLabs/nitro/blob/01412b3cd0fca28bf9931407ca1ccfeb8714d478/contracts/src/bridge/Inbox.sol#L262)
  - For the redemption of retryable tickets, the calculation of the L2 transaction ID changed and differs between different redeem attempts (i.e. after failed attempts). See [arbitrum-sdk](https://github.com/offchainlabs/arbitrum-sdk/tree/c-nitro-stable) for a reference implementation on the new client-side flow.
  - A retryable ticket now isn't redeemed in the same transaction as when the `redeem` function was called. The user's transaction causes the retryable to be scheduled to be executed after the current transaction is complete. More information on this available in [here](../arbos/arbos.md#redeeming-a-retryable).
  - Auto-redeem will not be created if the user does not have enough balance to pay for `gasFeeCap * gasLimit` (meaning you can no longer set a max gas fee cap).
  - Deposited gas will be refunded to `excessFeeRefundAddress` if it cannot create an auto-redeem.
  - The user will be refunded the submission cost of their retryable if it is auto-redeemed.
  - The lifecycle of retryable tickets are now tracked differently. Previously there was a retryable ticket ID, which could be used to deterministically generate the expected tx hash. In Nitro, you instead have a retryable creation tx hash (which can be retrieved by the SDK's `L1ToL2Message.retryableCreationId` or calculated by calling [L1ToL2Message.calculateSubmitRetryableId](https://github.com/OffchainLabs/arbitrum-sdk/blob/105bf73cb788231b6e63c510713f460b36699fcd/src/lib/message/L1ToL2Message.ts#L109-L155)). This value does not directly map into an expected tx hash where it was redeemed. You need to instead listen to the [RedeemScheduled](https://github.com/OffchainLabs/nitro/blob/ec70ed7527597e7e1e8380a59c07e8449885e408/contracts/src/precompiles/ArbRetryableTx.sol#L85-L93) event, which tells you the expected `retryTxHash` of that attempt.
  - The retryTxHash is no longer deterministic solely based on the retryable ticket id; it is now a hash of the transaction input like a normal transaction (following the [Typed Tx Envelope standard](https://eips.ethereum.org/EIPS/eip-2718))
  - If the retryable `to` address is set to the zero address, it will now function as a contract deployment.
- **Arbitrum blockhash**: `blockhash(x)` returns a cryptographically insecure, pseudo-random hash for `x` within the range `block.number - 256 <= x < block.number`. If `x` is outside of this range, `blockhash(x)` will return `0`. This includes `blockhash(block.number)`, which always returns `0` just like on Ethereum. The hashes returned do not come from L1.
- **ArbSys precompile**: `ArbSys.getTransactionCount` and `ArbSys.getStorageAt` are removed in nitro

#### Protocol Contracts

- **New Contract Deployments**: For the Nitro upgrade, these contracts were redeployed on L1 to new addresses:
  - SequencerInbox
  - RollupCore
  - Outbox
  - Bridge
    - bridge.sol contract will be redeployed and not the whole Bridge.
    - The [bridge contract address](https://etherscan.io/address/0x011B6E24FfB0B5f5fCc564cf4183C5BBBc96D515) was be changed.

For addresses of protocol contracts, see [Useful Addresses](../useful-addresses.md).

Also, worth mentioning that the address of the Delayed Inbox contract and Token Bridge contracts (Router and Gateways) weren't changed after the migration.

- **Sequencer Inbox changes**: The Sequencer inbox has a new interface and requires a new approach to determining a transaction's inclusion on L1 (see "Batch Info In Receipts" below).

- **Outbox Changes**: The Outbox has a new (simplified!) architecture; in short, all outgoing messages will be included in a single Merkle tree (opposed to Arbitrum classic, in which many outbox entries, each with its own Merkle root). See [arbitrum-sdk](https://github.com/offchainlabs/arbitrum-sdk/tree/c-nitro) for a reference implementation on how to handle the new flow of interacting with the outbox.

#### RPCs

- **No Parity tracing**: Initially `trace_filter` RPCs won't be available; they will be in the coming months. (Note that the Geth tracing APIs _are_ available).

- **Gas Info in Transaction Receipts**: Arbitrum transaction receipts return data about gas in a new format; receipts will have `gasUsed` (total) and `gasUsedForL1` fields (instead of the `feeStats` field in Arbitrum classic).

- **Batch Info In Receipts**: Arbitrum transaction receipts no longer include the `l1SequenceNumber` field; the `findBatchContainingBlock` or `getL1Confirmations` methods in the [NodeInterface precompile](../../contracts/src/node-interface/NodeInterface.sol) can be used to determine a transaction's inclusion in L1.

- **Estimate Retryable Ticket**: Use `eth_estimateGas` on `NodeInterface.estimateRetryableTicket` to estimate the gas limit of a retryable; the function itself no longer return the gas used and gas price. The gas price can be estimated using `eth_gasPrice`.
