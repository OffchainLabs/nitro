### Fixed
- Prevent unintended mutation of `latestHeader.Number` in `ParentChainIsUsingEIP7623` by using `new(big.Int).Sub()` instead of calling `Sub` directly on the header's number field
