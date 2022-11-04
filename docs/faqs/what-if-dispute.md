# FAQ: Delays Disputes and Reorgs

## If there is a dispute, can my L2 transaction get reorged / throw out / "yeeted"? ?

Nope; once an Arbitrum transaction is included on L1, there is no way it can be reorged (unless the L1 itself reorgs, of course). A "dispute" involves Validators disagreeing over execution, i.e., the outputted state of a chain. The inputs, however, can't be disputed; they are determined by the Inbox on L1. (See [Transaction Lifecycle](../tx-lifecycle.md))

## ...okay but if there's a dispute, will my transaction get delayed?

The only thing that a dispute can add delay to is the confirmation of L2-to-L1 messages. All other transactions continue to be processed, even while a dispute is still undergoing. (Additionally: in practice, most L2 to L1 messages represent withdrawals of fungible assets; these can be trustlessly completed _even during a dispute_ via trustless fast "liquidity exit" applications. See [L2 To L1 Messages](../arbos/l2-to-l1-messaging.md)).

## ...okay okay, but if we're just talking about an L2-to-L1 message, and assuming there's no disputes, how long between the time the message is initiated and when I can execute it on L1? Is it exactly one week?

It will be roughly one week, but there's some variability in the exact wall-clock time of the dispute window, plus there's some expected additional "padding" time on both ends (no more than about an hour, typically).

The variability of the dispute window comes from the slight variance of block times. Arbitrum One's dispute window is 45818 blocks; this converts to ~1 week assuming 13.2 seconds per block, which was the average block time when Ethereum used Proof of Work (with the switch to Proof of Stake, average block times are expected to be slightly lower — about 12 seconds.)

The "padding on both ends" involves three events that have to occur between a client receiving their transaction receipt from a Sequencer and their L2 to L1 message being executable. After getting their receipt,

1. The Sequencer posts their transaction in a batch (usually within a few minutes, though the Sequencer will wait a bit longer if the L1 is congested). Then,
1. A validator includes their transaction in an RBlock (usually within the hour).
   Then, after the ~week long dispute window passes, the RBlock is confirmable, and
1. Somebody (anybody) confirms the RBlock on L1. (usually within ~15 minutes)
