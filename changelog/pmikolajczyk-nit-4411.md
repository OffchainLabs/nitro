### Configuration
 - Extend genesis.json with `serializedConfig` and `arbOSInit.initialL1BaseFee` fields.
 - Remove `initial-l1-base-fee` CLI flag from genesis-generator

### Changed
 - genesis-generator will now firstly try to read init message data directly from genesis.json
