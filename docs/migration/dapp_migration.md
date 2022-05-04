# Nitro Migration Notes for Solidity Devs: Living Document


The following is a summary of changes in the upcoming Arbitrum One chain's Nitro upgrade that dapp devs should be aware of. 

It reflects the current state of Nitro and should be considered a living document; things might change before launch.

_Last Updated 4/13/2022_

## Cool New Stuff 

For starters, here's a sampling of exciting perks dapps with get with the Nitro upgrade:

- **Ethereum L1 Gas Compatibility 🥳**:  gas pricing and accounting for EVM operations is be perfectly in line with L1; no more ArbGas.  
- **Safer Retryable tickets 🥳**: Retryable tickets' submission cost is collected in the L1 Inbox contract; if the submission cost is too low, the transaction will simply revert on the L1 side, eliminating the [failure mode](https://developer.offchainlabs.com/docs/l1_l2_messages#important-note-about-base-submission-fee) in which a retryable ticket fails to get created. 
- **Calldata compression 🥳**: Compression takes place protocol level; dapps don't need to change anything, data will just get cheaper! (You are charged even less if your calldata is highly compressible with brotli.)
- **Support for All Ethereum L1 precompiles 🥳**: (`blake2f`, `ripemd160`, etc)
- **Tighter Syncronization with L1 Block Numbers / Timestamps 🥳**:  L1 block number and timestamps (accessed via `block.number` and `block.timestamp` on L2) are updated more frequently in Nitro than in Arbitrum classic; expect them to be nearly real-time/ in sync with L1. 

- **L2 Block hash EVM Consistency 🥳**: L2 block hashes take the same format as on Ethereum (if you query it from the ArbSys precompile, not the one in `block.hash(uin256)`).


- **Geth tracing 🥳**: `debug_traceTransaction` RPC endpoint is supported; this includes tracing of ArbOS internal bookkeeping actions.

## Breaking changes

#### Dapps
- **Gas Accounting**: it is now consistent with the L1 EVM, any hard-coded gas values should be changed accordingly (the same applies to any gas amount used in conjuntion with `gasleft`).
- **No more storage gas**: there is no more concept of a separate pool of storage gas, opcodes are prices identically to the L1 EVM.
- **Retryable Tickets**: 
    - The submission cost is now enforced in the L1 inbox and checked against the L1 transaction's `msg.value`; contracts shouldn't rely on funds pooled in the L2 destination to cover this cost.
    - For the redemption of retryable tickets, the calculation of the L2 transaction ID changed, as has the transaction lifecycle of attempting multiple redemptions (i.e., after failed attempts). See [arbitrum-sdk](https://github.com/OffchainLabs/arbitrum-sdk) for a reference implementation on the new client-side flow. 

#### Protocol Contracts 

- **New Contract Deployments**: For the Nitro upgrade, the following contracts will be redeployed on L1 to new addresses:
    - SequencerInbox
    - RollupCore
    - Outbox

- **Sequencer Inbox changes**: The Sequencer inbox has a new interface and requires a new approach to determining a transaction's inclusion on L1 (see "Batch Info In Receipts" below).


- **Outbox Changes**: The Outbox has a new (simplified!) architecture; in short, all outgoing messages will be included in a single Merkle tree (opposed to Arbitrum classic, in which many outbox entries, each with its own Merkle root). See [arbitrum-sdk](https://github.com/OffchainLabs/arbitrum-sdk) for a reference implementation on how to handle the new flow of interacting with the outbox.

#### RPCs

- **No Parity tracing**: Initially `trace_filter` RPCs won't be available; they will be in the coming months. (Note that the Geth tracing APIs _are_ available).

- **Gas Info in Transaction Receipts**: Arbitrum transaction receipts return data about gas in a new format; receipts will have `gasUsed` (total) and `gasUsedForL1` fields (instead of the `feeStats` field in Arbitrum classic).

- **Batch Info In Receipts**: Arbitrum transaction receipts no longer include the `l1SequenceNumber` field; the `findBatchContainingBlock` or `getL1Confirmations` methods in the [NodeInterface precompile](../arbos/Precompiles.md) can be used to determine a transaction's inclusion in L1.
