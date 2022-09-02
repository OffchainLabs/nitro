# The Assertion Tree

### Overview

The state of an Arbitrum chain is confirmed back on Ethereum via "assertions," aka "disputable assertions" or "DAs." These are claims made by Arbitrum validators about the chain's state. To make an assertion, a validator must post a bond in Ether.

In the happy / common case, all outstanding assertions will be valid; i.e., a valid assertion will build on another valid assertion, which builds on another valid assertion, and so on. After the dispute period (~ 1 week) passes and an assertion goes unchallenged, it can be confirmed back on L1.

If, however, two or more conflicting assertions exist, the Assertion Tree bifurcates into multiple branches:

![img](assertionTree.png)

Crucially, the rules of advancing an Arbitrum chain are deterministic; this means that given a chain state and some new inputs, there is only one valid output. Thus, if the Assertion Tree contains more than one leaf, then at most only one leaf can represent the valid chain-state; if we assume there is at least one honest active validator, _exactly_ one leaf will be valid.

Two conflicting assertions can be put into a dispute; see [Interactive Challenges](./proving/challenge-manager.md) for details on the dispute process. For the sake of understanding the Assertion Tree protocol, suffice it to say that 2-party disputes last at most a fixed amount of time (1 week), at the end of which one of the two conflicting assertions will be rejected, and the validator who posted it will lose their stake.

In order for an assertion to be confirmed and for its stake to be recovered, two conditions must be met: sufficient time for disputes must have passed, and no other conflicting branches in the Assertion Tree can exist (i.e., they've all been disputed / "pruned" off.)

These properties together ensure that as long as at least one honest, active validator exists, the valid chain state will ultimately be confirmed.

### Delays

Even if the Assertion Tree has multiple conflicting leaves and, say, multiple disputes are in progress, validators can continue making assertions; honest validators will simply build on the one valid leaf (intuitively: an assertion is also an implicit claim of the validity of all of its parent-assertions.) Likewise, users can continue transacting on L2, since transactions continue to be posted in the chain's inbox.

The only delay that users experience during a dispute is of their [L2 to L1 messages](./arbos/l2-to-l1-messaging.md) (i.e., "their withdrawals"). Note that a "delay attacker" who seeks to grief the system by deliberately causing such delays will find this attack quite costly, since each bit of delay-time gained requires the attacker lose another stake.

### Detailed Spec

For a more detailed breakdown / specification of the assertion tree protocol, see [Inside Arbitrum](inside-arbitrum-nitro#arbitrum#rollup#protocol).
