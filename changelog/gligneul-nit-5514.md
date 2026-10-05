### Fixed

- The sequencer now reads the sender's nonce from the post-execution state instead of assuming every transaction advances it by one. A self-sponsored EIP-7702 transaction advances the nonce by two, and the stale value made the sequencer reject a following transaction from the same sender in the same block with `nonce too high`.
