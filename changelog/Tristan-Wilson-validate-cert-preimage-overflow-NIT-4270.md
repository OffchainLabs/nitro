### Fixed
- Fixed ValidateCertificate proof generation to panic on preimageType overflow (> 255) instead of silently using a fallback value, aligning with the Solidity one-step prover which reverts for this case.
