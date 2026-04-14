### Fixed
- Check ArbOS version in `FreeAccessPrecompile.Call` before reading state, matching the gate in `Precompile.Call`. Fixes consensus divergence on blocks that CALLed 0x74 prior to the activation of ArbFilteredTransactionsManager, where the wrapper's filterer lookup burned gas that the canonical chain did not charge.
