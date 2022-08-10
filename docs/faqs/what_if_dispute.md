# If there is a dispute, can my L2 transaction get reorged/ throwned out?

Nope; once an Arbitrum transaction is included on L1, there is no way it can be reorged (unless the L1 itself reorgs, of course). A "dispute" involves Validators disagreeing over execution, i.e., the outputted state of a chain. The inputs, however, can't be disputed; they are determined by the Inbox on L1. (See "Transaction Lifecycle")

## ...okay but if there's a dispute, will my transaction get delayed?

The only thing that a dispute can add delay to is the confirmation of L2-to-L1 messages. All other transactions continue to be processed, even while a dispute is still undergoing. (Additionally: in practice, most L2 to L1 messages represent withdrawals of fungible assets; these can be trustlessly completed _even during a dispute_ via trustless fast "liquidity exit" applications. See "L2 To L1 Messages.")
