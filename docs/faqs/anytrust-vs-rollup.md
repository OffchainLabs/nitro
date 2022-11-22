# Q: What's the difference between Arbitrum Rollup and Arbitrum AnyTrust?

**A:** Arbitrum Rollup is an Optimistic Rollup protocol; it is trustless and permissionless. Part of how these properties are achieved is by requiring all chain data to be posted on layer 1. This means the availability of this data follows directly from the security properties of Ethereum itself, and, in turn, that any party can participate in validating the chain and ensuring its safely.

By contrast, Arbitrum AnyTrust introduces a trust assumption in exchange for lower fees; data availability is managed by a Data Availability Committee (DAC), a fixed, permissioned set of entities. We introduced some threshold, K, with the assumption that at least K members of the committee are honest. For simplicity, we'll hereby assume a committee of size 20 and a K value of 2:

If 19 out of the 20 committee members _and_ the Sequencer are malicious and colluding together, they can break the chain's safety (and, e.g., steal users' funds); this is the new trust assumption.

If anywhere between 2 and 18 of the committee members are well behaved, the AnyTrust chain operates in "Rollup mode"; i.e., data gets posted on L1.

In what should be the common and happy case, however, in which at least 19 of the 20 committee members are well behaved, the system operates without the posting the L2 chain's data on L1, and thus, users pay significantly lower fees. This is the core upside of AnyTrust chains over rollups.

Variants of the AnyTrust model in which the new trust assumption is minimized are under consideration; stay tuned.

For more, see [Inside AnyTrust](../inside-anytrust.md).
