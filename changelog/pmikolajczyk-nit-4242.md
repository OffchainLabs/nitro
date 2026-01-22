### Configuration
- Extend the genesis configuration with `initial-l1base-fee` and `serialized-chain-config` fields.
- Enable patching these two parameters with node-level CLI flags: `--init.genesis-patch.initial-l1base-fee` and `--init.genesis-patch.serialized-chain-config`.
- Remove `initial-l1-base-fee` flag from `genesis-generator` tool.
