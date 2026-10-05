### Fixed
- BoLD auto-deposit now detects WETH-compatible stake tokens via an `eth_call` probe of `deposit()` instead of scanning the token's bytecode, so stake tokens deployed behind a proxy are no longer misclassified as non-WETH.
