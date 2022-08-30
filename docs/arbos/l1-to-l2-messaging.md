# L1 To L2 Messaging

## Retryables

Retryable tickets are Arbitrum's canonical method for creating L1 to L2 messages, i.e., L1 transactions that initiate a message to be executed on L2. A retryable can be submitted for a fixed cost (dependent only on its calldata size) paid at L1; its _submission_ on L1 is separable / asynchronous with its _execution_ on L2. Retryables provide atomicity between the cross chain operations; if the L1 transaction to request submission succeeds (i.e. does not revert) then the execution of the Retryable in L2 has a strong guaranteed to ultimately succeed as well.

A retryable is submitted via the `Inbox`'s `createRetryableTicket` method. In the common case, a Retryable's submission is followed by an attempt to execute the transaction (i.e., an _"auto-redeem"_). If the attempt fails or isn't scheduled after the Retryable is submitted, anyone can try to _redeem_ it, by calling the [`redeem`](./precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](./precompiles.md#ArbRetryableTx) precompile. The party requesting the redeem provides the gas that will be used to execute the Retryable. If execution of the Retryable succeeds, the Retryable is deleted. If execution fails, the Retryable continues to exist and further attempts can be made to redeem it. If a fixed period (currently one week) elapses without a successful redeem, the Retryable expires and will be [automatically discarded][discard_link], unless some party has paid a fee to [_renew_][renew_link] the Retryable for another full period. A Retryable can live indefinitely as long as it is renewed each time before it expires.

[discard_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/retryables/retryable.go#L262
[renew_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/retryables/retryable.go#L207

### Submitting a Retryable

A transaction to submit a Retryable does the following:

- create a new Retryable with the caller, destination, callvalue, calldata, and beneficiary of the submit transaction
- deduct funds to cover the callvalue from the caller (as usual) and place them into escrow for later use in redeeming the Retryable
- assign a unique TicketID to the Retryable
- cause the ArbRetryableTx precompiled contract to emit a TicketCreated event containing the TicketID
- if the submit transaction contains gas, schedule a redeem of the new Retryable, using the supplied gas, as if the [`redeem`](./precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](./precompiles.md#ArbRetryableTx) precompile had been called.

In most use cases, the submitter will provide gas and will intend for the immediate redeem to succeed, with later retries available only as a backup mechanism should the immediate redeem fail. (It might fail, for example, because the L2 gas price has increased unexpectedly.) In this way, an L1 contract can submit a transaction to L2 in such a way that the transaction will normally run immediately at L2 but allowing any party to retry the transaction should it fail.

When a Retryable is redeemed, it will execute with the sender, destination, callvalue, and calldata of the original submission. The callvalue will have been escrowed during the initial submission of the Retryable, for this purpose. If a Retryable with callvalue is eventually discarded, having never successfully run, the escrowed callvalue will be paid out to a "beneficiary" account that is specified in the initial submission.

A Retryable's beneficiary has the unique power to [`cancel`](./precompiles.md#ArbRetryableTx) the Retryable. This has the same effect as the Retryable timing out.

[moved_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L191

### Redeeming a Retryable

If a redeem is not done at submission or the submission's initial redeem fails, anyone can attempt to redeem the retryable again by calling [`ArbRetryableTx`](precompiles.md#ArbRetryableTx)'s [`redeem`](./precompiles.md#ArbRetryableTx) precompile method, which donates the call's gas to the next attempt. ArbOS will [enqueue the redeem][enqueue_link], which is its own special `ArbitrumRetryTx` type, to its list of redeems that ArbOS [guarantees to exhaust][exhaust_link] before moving on to the next non-redeem transaction in the block its forming. In this manner redeems are scheduled to happen as soon as possible, and will always be in the same block as the tx that scheduled it. Note that the redeem attempt's gas comes from the call to [`redeem`](./precompiles.md#ArbRetryableTx), so there's no chance the block's gas limit is reached before execution.

On success, the `To` address keeps the escrowed callvalue, and any unused gas is returned to ArbOS's gas pools. On failure, the callvalue is returned to the escrow for the next redeemer. In either case, the network fee was paid during the scheduling tx, so no fees are charged and no refunds are made.

During redemption of a retryable, attempts to cancel the same retryable, or to schedule another redeem of the same retryable, will revert. In this manner retryables are not self-modifying.

[enqueue_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L245
[exhaust_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L135

### Receipts

If the lifecycle of a retryable ticket, two types of L2 transaction receipts will be emitted:

**Ticket Creation Receipt**: This receipts indicates that a retryable ticket was successfully created; any successful L1 call to the `Inbox`'s `createRetryableTicket` method is guaranteed to create a ticket. The ticket creation receipt includes a `TicketCreated` event (from [`ArbRetryableTx`](./precompiles.md#ArbRetryableTx)), which includes a `ticketId` field. This `ticketId` is computable via RLP encoding and hashing the transaction; see [calculateSubmitRetryableId](https://github.com/OffchainLabs/arbitrum-sdk/blob/6cc143a3bb019dc4c39c8bcc4aeac9f1a48acb01/src/lib/message/L1ToL2Message.ts#L109).

**Redeem Attempt**: A redeem attempt receipt represents the result of an attempted L2 execution of a retryable ticket. It includes a `RedeemScheduled` event from [`ArbRetryableTx`](./precompiles.md#ArbRetryableTx), with a `ticketId` field. At most, one successful redeem attempt can ever exist for a given ticket; if, e.g., the auto-redeem upon initial creation succeeds, only the receipt from the auto-redeem will ever get emitted for that ticket. If the auto-redeem fails (or was never attempted — i.e., the provided L2 gas limit \* L2 gas price = 0), each initial attempt will emit a redeem attempt receipt until one succeeds.


### Alternative "unsafe" Retryable Ticket Creation

The `Inbox.createRetryableTicket` convenience method includes sanity checks to help minimize the risk of user error: the method will ensure that enough funds are provided directly from L1 to cover the current cost of ticket creation / execution. It also will convert the provided `beneficiary` or `credit-back-address` to its address alias (see below) if either is a contract, providing a path for the L1 contract to recover funds. A power-user may by-pass these sanity-check measures via the `Inbox`'s `unsafeCreateRetryableTicket` method; as the method's name desperately attempts to warn you, it should only be accessed by a user who truly knows what they're doing. 

## Eth deposits

A special message type exists for simple Eth deposits; i.e., sending Eth from L1 to L2. Eth can be deposited via a call to the `Inbox`'s `depositEth` method. If the L1 caller is EOA, the Eth will be deposited to the same EOA address on L2; the L1 caller is a contract, the funds will deposited to the contract's aliased address (see below).

Note that depositing Eth via `depositEth` into a contract on L2 will _not_ trigger the contract's fallback function.

In principle, retryable tickets can alternatively be used to deposit Ether; this could be preferable to the special eth-deposit message type if, e.g., more flexibility for the destination address is needed, or if one wants to trigger the fallback function on the L2 side.

## Transacting via the Delayed Inbox

While Retryables and Eth deposits _must_ be submitted through the delayed inbox, in principle, _any_ message can be included this way; this is a necessary recourse to ensure the Arbitrum chain preserves censorship resistance even if the Sequencer misbehaves (see [The Sequencer and Censorship Resistance](../sequencer.md)). However, under ordinary/happy circumstances, the expectation/recommendation is that clients use the delayed inbox only for Retryables and Eth deposits, and transact via the Sequencer for all other messages.

## Address Aliasing

All messages submitted via the Delayed Inbox get their sender's addressed "aliased": when these unsigned messages are executed on L2, the sender's address —i.e., that which is returned by `msg.sender` — will _not_ simply be the L1 address that sent the message; rather it will be the address's "L2 Alias." An address's L2 alias is its value increased by the hex value `0x1111000000000000000000000000000000001111`:

```
L2_Alias = L1_Contract_ Address + 0x1111000000000000000000000000000000001111
```

The Arbitrum protocol's usage of L2 Aliases for L1-to-L2 messages prevents cross-chain exploits that would otherwise be possible if we simply reused the same L1 addresses as the L2 sender; i.e., tricking an L2 contract that expects a call from a given contract address by sending retryable ticket from the expected contract address on L1. 

If for some reason you need to compute the L1 address from an L2 alias on chain, you can use our `AddressAliasHelper` library:

```sol
    modifier onlyFromMyL1Contract() override {
        require(AddressAliasHelper.undoL1ToL2Alias(msg.sender) == myL1ContractAddress, "ONLY_COUNTERPART_CONTRACT");
        _;
    }
```
