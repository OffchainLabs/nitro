### Internal
- Decompose the nitro-reth block producer: extract flush scheduling, genesis init, the block-gas-limit predicate, internal-tx helpers, and bundle post-processing out of `producer.rs`; block production takes the canonical `MessageWithMetadata` instead of the duplicate `BlockProductionInput`.
