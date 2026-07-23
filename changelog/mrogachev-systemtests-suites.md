### Internal
- Reorganize v2 system tests into per-suite packages (precompiles, gas, arbos, sequencer, messaging, rpc, validation) and port a curated scenario set onto the framework, using matrix expansion to collapse per-version test copies (precompile inclusion across 5 ArbOS versions, block difficulty across 2) into single parameterized bodies; fully ported v1 tests are removed.
