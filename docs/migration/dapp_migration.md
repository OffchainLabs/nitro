# Nitro Migration Notes for Solidity Devs: Living Document


The following is a summary of changes in the upcoming Arbitrum One chain's Nitro upgrade that dapp devs should be aware of. 

It reflects the current state of Nitro and should be considered a living document; things might change before launch.

_Last Updated 5 Aug 2022_

## Cool New Stuff 

For starters, here's a sampling of exciting perks dapps with get with the Nitro upgrade:

- **Ethereum L1 Gas Compatibility 🥳**:  gas pricing and accounting for EVM operations is be perfectly in line with L1; no more ArbGas.  

- **Safer Retryable tickets 🥳**: Retryable tickets' submission cost is collected in the L1 Inbox contract; if the submission cost is too low, the transaction will simply revert on the L1 side, eliminating the [failure mode](https://developer.offchainlabs.com/docs/l1_l2_messages#important-note-about-base-submission-fee) in which a retryable ticket fails to get created. 

- **Calldata compression 🥳**: Compression takes place protocol level; dapps don't need to change anything, data will just get cheaper! (You are charged even less if your calldata is highly compressible with brotli.)

- **Support for All Ethereum L1 precompiles 🥳**: (`blake2f`, `ripemd160`, etc)

- **Tighter Syncronization with L1 Block Numbers 🥳**:  L1 block number (accessed via `block.number` on L2) are updated more frequently in Nitro than in Arbitrum classic; expect them to be nearly real-time/ in sync with L1. 

- **Frequent Timestamps 🥳**:  Timestamps (accessed via `block.timestamp` on L2) are updated every block based on the sequencer’s clock, it is no longer linked to the timestamp of the last L1 block.

- **L2 Block hash EVM Consistency 🥳**: L2 block hashes take the same format as on Ethereum (if you query it from the ArbSys precompile, not the one in `block.hash(uin256)`).

- **Geth tracing 🥳**: `debug_traceTransaction` RPC endpoint is supported; this includes tracing of ArbOS internal bookkeeping actions.

## Breaking changes

#### Dapps

- **Gas Accounting**: it is now consistent with the L1 EVM, L2 gas usage will change due to different accounting from ArbGas. Any hard-coded gas values should be changed accordingly (the same applies to any gas amount used in conjuntion with `gasleft`). That said, you shouldn't be hard-coding any gas values just like in Ethereum, since both the L1 and L2 gas schedule may change in the future.

- **No more storage gas**: there is no more concept of a separate pool of storage gas, and opcodes are priced identically to the L1 EVM.

- **New L2 to L1 event signature**: The function signature for the [L2 to L1 event](../../contracts/src/precompiles/ArbSys.sol#L110) emitted by ArbSys has now changed.

- **Lower contract code size limit**: Contracts of up to 48KB were deployable, but now only up to 24KB are deployable (as specified in [EIP 170](https://eips.ethereum.org/EIPS/eip-170)). Previously deployed contracts above the limit will be maintained (but contracts deployed by these legacy contracts are capped by the new size).

- **Retryable Tickets**: 
    - The submission cost is now enforced in the L1 inbox and checked against the L1 transaction's `msg.value`; contracts shouldn't rely on funds pooled in the L2 destination to cover this cost.
    - The current submission price is now not available in the L2 ArbRetryableTx precompile, instead it can be queried in the L1 Delayed Inbox [`calculateRetryableSubmissionFee(uint256 dataLength, uint256 baseFee)`](https://github.com/OffchainLabs/nitro/blob/01412b3cd0fca28bf9931407ca1ccfeb8714d478/contracts/src/bridge/Inbox.sol#L262)
    - For the redemption of retryable tickets, the calculation of the L2 transaction ID changed and differs between different redeem attempts (i.e. after failed attempts). See [arbitrum-sdk](https://github.com/offchainlabs/arbitrum-sdk/tree/c-nitro-stable) for a reference implementation on the new client-side flow. 
    - A retryable ticket now isn't redeemed in the same transaction as when the `redeem` function was called. The user's transaction causes the retryable to be scheduled to be executed after the current transaction is complete. More information on this available in [here](../arbos/ArbOS.md#redeeming-a-retryable).
    - Auto-redeem will not be created if the user does not have enough balance to pay for `gasFeeCap * gasLimit` (meaning you can no longer set a max gas fee cap).
    - Deposited gas will be refunded to `excessFeeRefundAddress` if it cannot create an auto-redeem.
    - The user will be refunded the submission cost of their retryable if it is auto-redeemed.
    - The retryable ticket id is now the retryable creation tx hash; with arbitrum sdk you can retrieve it with `L1ToL2Message.retryableCreationId` or calculate it by calling [L1ToL2Message.calculateSubmitRetryableId](https://github.com/OffchainLabs/arbitrum-sdk/blob/105bf73cb788231b6e63c510713f460b36699fcd/src/lib/message/L1ToL2Message.ts#L109-L155)
    - The retryable redemption hash is no longer deterministic solely based on the retryable ticket id; it is now a hash of the transaction input like a normal transaction 

#### Protocol Contracts 

- **New Contract Deployments**: For the Nitro upgrade, the following contracts will be redeployed on L1 to new addresses:
    - SequencerInbox
    - RollupCore
    - Outbox

- **Sequencer Inbox changes**: The Sequencer inbox has a new interface and requires a new approach to determining a transaction's inclusion on L1 (see "Batch Info In Receipts" below).

- **Outbox Changes**: The Outbox has a new (simplified!) architecture; in short, all outgoing messages will be included in a single Merkle tree (opposed to Arbitrum classic, in which many outbox entries, each with its own Merkle root). See [arbitrum-sdk](https://github.com/offchainlabs/arbitrum-sdk/tree/c-nitro) for a reference implementation on how to handle the new flow of interacting with the outbox.

#### RPCs

- **No Parity tracing**: Initially `trace_filter` RPCs won't be available; they will be in the coming months. (Note that the Geth tracing APIs _are_ available).

- **Gas Info in Transaction Receipts**: Arbitrum transaction receipts return data about gas in a new format; receipts will have `gasUsed` (total) and `gasUsedForL1` fields (instead of the `feeStats` field in Arbitrum classic).

- **Batch Info In Receipts**: Arbitrum transaction receipts no longer include the `l1SequenceNumber` field; the `findBatchContainingBlock` or `getL1Confirmations` methods in the [NodeInterface precompile](../../contracts/src/node-interface/NodeInterface.sol) can be used to determine a transaction's inclusion in L1.

- **Estimate Retryable Ticket**: Use `eth_estimateGas` on `NodeInterface.estimateRetryableTicket` to estimage the gas limit of a retryable; the function itself no longer return the gas used and gas price. The gas price can be estimated using `eth_gasPrice`.
