# Orbit with Celestia Underneath ✨
![image](https://github.com/celestiaorg/nitro/assets/31937514/dfe451b5-21ee-446b-8140-869ea4e2a7eb)


## Overview

The integration of Celestia with Arbitrum Orbit and the Nitro tech stack marks the first external contribution to the Arbitrum Orbit protocol layer, offering developers an additional option for selecting a data availability layer alongside Arbitrum AnyTrust. The integration allows developers to deploy an Orbit Chain that uses Celestia for data availability and settles on Arbitrum One, Ethereum, or other EVM chains.

## Key Components

The integration of Celestia with Arbitrum orbit is possible thanks to 3 components:
- DA Provider Implementation
- Preimage Oracle
- Blobstream

# DA Provider Implementation

The Arbitrum Nitro code has a `DataAvailabilityProvider` interface that is used across the codebase to store and retrieve data from a specific provider (eip4844 blobs, Anytrust, and now Celestia).

This integration implements the [`DataAvailabilityProvider` interface for Celestia DA](https://github.com/celestiaorg/nitro/blob/966e631f1a03b49d49f25bea67a92b275d3bacb9/arbstate/inbox.go#L366-L477)

Additionally, this integrations comes with the necessary code for a Nitro chain node to post and retrieve data from Celestia, which can be found [here.](https://github.com/celestiaorg/nitro/tree/celestia-v2.3.1/das/celestia)

The core logic behind posting and retrieving data happens in [celestia.go](https://github.com/celestiaorg/nitro/blob/celestia-v2.3.1/das/celestia/celestia.go) where data is stored on Celestia and serialized into a small batch of data that gets published once the necessary range of headers (data roots) has been relayed to the [BlobstreamX contract](https://github.com/succinctlabs/blobstreamx).
Then the `Read` logic takes care of taking the deserialized Blob Pointer struct and consuming it in order to fetch the data from Celestia and additionally inform the fetcher about the position of the data on Celestia (we'll get back to this in the next section).

The following represents a non-exhaustive list of considerations when running a Batch Poster node for a chain with Celestia underneath:
- You will need to use a consensus full node RPC endpoint, you can find a list of them for Mocha [here](https://docs.celestia.org/nodes/mocha-testnet#rpc-endpoints)
- The Batch Poster will only post a Celestia batch to the underlying chain if the height for which it posted is in a recent range in BlobstreamX and if the verification succeeds, otherwise it will discard the batch. Since it will wait until a range is relayed, it can take several minutes for a batch to be posted, but one can always make an on-chain request for the BlobstreamX contract to relay a header promptly.
- 

The following represents a non-exhaustive list of considerations when running a Nitro node for a chain with Celestia underneath:
- The `TendermintRpc` endpoint is only needed by the batch poster, every other node can operate without a connection to a full node.
- The message header flag for Celestia batches is `0x0c`.
- You will need to know the namespace for the chain that you are trying to connect to, but don't worry if you don't find it, as the information in the BlobPointer can be used to identify where a batch of data is in the Celestia Data Square for a given height, and thus can be used to find out the namespace as well!

# Preimage Oracle Implementation

In order to support fraud proofs, this integration has the necessary code for a Nitro validator to pupolate its preimage mapping with Celestia hashes that then get "unpealed" in order to reveal the full data for a Blob. You can read more about the "Hash Oracle Trick" [here.](https://docs.arbitrum.io/inside-arbitrum-nitro/#readpreimage-and-the-hash-oracle-trick)

The data structures and hashing functions for this can be found in the [`nitro/das/celestia/tree` folder](https://github.com/celestiaorg/nitro/tree/celestia-v2.3.1/das/celestia/tree)

You can see where the preimage oracle gets used in the fraud proof replay binary [here](https://github.com/celestiaorg/nitro/blob/966e631f1a03b49d49f25bea67a92b275d3bacb9/cmd/replay/main.go#L153-L294)

Something important to note is that the preimage oracle only keeps track of hashes for the rows in the Celestia data square in which a blob resides in, this way each Orbit chain with Celestia underneath does not need validators to recompute an entire Celestia Data Square, but instead, only have to compute the row roots for the rows in which it's data lives in, and the header data root, which is the binary merkle tree hash built using the row roots and column roots fetched from a Celestia node. Because only data roots that can be confirmed on Blobstream get accepted into the sequencer inbox, one can have a high degree of certainty that the canonical data root being unpealed as well as the row roots are in fact correct.

# DA Proof and BlobstreamX 

Finally, the integration only accepts batches of 89 bytes in length for a celestia header flag. This means that a Celestia Batch has 88 bytes of information, which are the block height, the start index of the blob, the length in shares of the blob, the transaction commitment, and the data root for the given height.

In the case of a challenge, for a celestia batch, the OSP will require an additionally appended "da proof", which is verified against BlobstreamX. Here's what happens based on the result of the BlobstreamX verification:

- **IN_BLOBSTREAM**: means the batch was verified against blobstrea, the height and data root in the batch match, and the start + legth do not go out of bounds. This will cause the rest of the OSP to proceed as normal.
- **COUNTERFACTUAL_COMMITMENT**: the height can be verified against blobstream, but the posted data root does not match, or the start + length go out of bounds. Or the Batch Poster tried posting a height too far into the ftureu (1000 blocks ahead of BlobstreamX). This will cause the OSP to proceed with an empty batch. Note that Nitro nodes for a chain with Celestia DA will also discard any batches that cannot be correctly validated.
- **UNDECIDED**: the height has not been relayed yet, so we revert and wait until the latest height in blobstream is the greater than the batch's height.

You can see how BlobstreamX is integrated into the `OneStepProverHostIO.sol` contract [here]([https://github.com/celestiaorg/nitro-contracts/blob/celestia-v1.2.1/src/bridge/SequencerInbox.sol#L584-L630](https://github.com/celestiaorg/nitro-contracts/blob/contracts-v1.2.1/src/osp/OneStepProverHostIo.sol#L301)), which allows us to discard batches with otherwise faulty data roots, thus giving us a high degree of confidence that the data root can be safely unpacked in case of a challenge.



