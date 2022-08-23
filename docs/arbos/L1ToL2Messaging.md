# L1 To L2 Messaging

## Retryables

Retryable tickets are Arbitrum's canonical method for creating L1 to L2 messages, i.e., L1 transactions that initiate a message to be executed on L2. A retryable can be submitted for a fixed cost (dependent only on its calldata size) paid at L1; its *submission* on L1 is separable / asynchronous with its *execution* on L2. Retryables provide atomicity between the cross chain operations; if the L1 transaction to request submission succeeds (i.e. does not revert) then the submission of the Retryable to the L2 state has a strong guaranteed to ultimately succeed as well.


In the common case, a Retryable's submission is followed by an attempt to execute the transaction (i.e., an *"auto-redeem"*). If the attempt fails or isn't scheduled after the Retryable is submitted, anyone can try to *redeem* it, by calling the [`redeem`](Precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx) precompile. The party requesting the redeem provides the gas that will be used to execute the Retryable. If execution of the Retryable succeeds, the Retryable is deleted. If execution fails, the Retryable continues to exist and further attempts can be made to redeem it. If a fixed period (currently one week) elapses without a successful redeem, the Retryable expires and will be [automatically discarded][discard_link], unless some party has paid a fee to [*renew*][renew_link] the Retryable for another full period. A Retryable can live indefinitely as long as it is renewed each time before it expires.

[discard_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/retryables/retryable.go#L262
[renew_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/retryables/retryable.go#L207


### Submitting a Retryable

A transaction to submit a Retryable does the following:

* create a new Retryable with the caller, destination, callvalue, calldata, and beneficiary of the submit transaction
* deduct funds to cover the callvalue from the caller (as usual) and place them into escrow for later use in redeeming the Retryable
* assign a unique TicketID to the Retryable
* cause the ArbRetryableTx precompiled contract to emit a TicketCreated event containing the TicketID
* if the submit transaction contains gas, schedule a redeem of the new Retryable, using the supplied gas, as if the [`redeem`](Precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx) precompile had been called.

In most use cases, the submitter will provide gas and will intend for the immediate redeem to succeed, with later retries available only as a backup mechanism should the immediate redeem fail. (It might fail, for example, because the L2 gas price has increased unexpectedly.) In this way, an L1 contract can submit a transaction to L2 in such a way that the transaction will normally run immediately at L2 but allowing any party to retry the transaction should it fail.

When a Retryable is redeemed, it will execute with the sender, destination, callvalue, and calldata of the original submission. The callvalue will have been escrowed during the initial submission of the Retryable, for this purpose. If a Retryable with callvalue is eventually discarded, having never successfully run, the escrowed callvalue will be paid out to a "beneficiary" account that is specified in the initial submission.

A Retryable's beneficiary has the unique power to [`cancel`](Precompiles.md#ArbRetryableTx) the Retryable. This has the same effect as the Retryable timing out.

[moved_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L191

### Redeeming a Retryable

If a redeem is not done at submission or the submission's initial redeem fails, anyone can attempt to redeem the retryable again by calling [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx)'s [`redeem`](Precompiles.md#ArbRetryableTx) precompile method, which donates the call's gas to the next attempt. ArbOS will [enqueue the redeem][enqueue_link], which is its own special `ArbitrumRetryTx` type, to its list of redeems that ArbOS [guarantees to exhaust][exhaust_link] before moving on to the next non-redeem transaction in the block its forming. In this manner redeems are scheduled to happen as soon as possible, and will always be in the same block as the tx that scheduled it. Note that the redeem attempt's gas comes from the call to [`redeem`](Precompiles.md#ArbRetryableTx), so there's no chance the block's gas limit is reached before execution.

On success, the `To` address keeps the escrowed callvalue, and any unused gas is returned to ArbOS's gas pools. On failure, the callvalue is returned to the escrow for the next redeemer. In either case, the network fee was paid during the scheduling tx, so no fees are charged and no refunds are made. 

During redemption of a retryable, attempts to cancel the same retryable, or to schedule another redeem of the same retryable, will revert. In this manner retryables are not self-modifying.

[enqueue_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L245
[exhaust_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L135

### Receipts

If the lifecycle of a retryable ticket, two types of L2 transaction receipts will be emitted:

**Ticket Creation Receipt**: This receipts indicates that a retryable ticket was successfully created; any successful L1 call to the `Inbox`'s `createRetryableTicket` method is guaranteed to create a ticket. The ticket creation receipt includes `TicketCreated` event (from [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx)), which includes a `ticketId` field. This `ticketId` is computable via RLP encoding and hashing the transaction; see [calculateSubmitRetryableId](https://github.com/OffchainLabs/arbitrum-sdk/blob/6cc143a3bb019dc4c39c8bcc4aeac9f1a48acb01/src/lib/message/L1ToL2Message.ts#L109).

**Redeem Attempt**: A redeem attempt receipt represents the result of an attempted L2 execution of a retryable ticket. It includes a `RedeemScheduled` event from [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx), with a `ticketId` field. At most, one successful redeem attempt can ever exist for a given ticket; if, e.g., the auto-redeem upon initial creation succeeds, only the receipt from the auto-redeem will ever get emitted for that ticket. If the auto-redeem fails (or was never attempted — i.e., the provided L2 gas limit * L2 gas price = 0), each initial attempt will emit a redeem attempt receipt until one succeeds.

###  Eth deposits
Delayed inbox

Address Aliasing
