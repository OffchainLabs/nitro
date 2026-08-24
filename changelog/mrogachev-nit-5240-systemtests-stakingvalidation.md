### Internal
- Add a staking-validation topology to the v2 system-tests framework: `WithStakingValidation()` builds an L1 + sequencer L2 plus a follower that runs block validation and stakes; post-hooks verify validation to head and a staked assertion.
