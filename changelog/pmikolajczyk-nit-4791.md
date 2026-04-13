### Changed
- Disable multi-value WASM for Stylus V3 contracts: upon activation at ArbOS version 60, the Stylus runtime version is upgraded to V3, and any new activation of a WASM containing multi-value blocks or functions is rejected. This replaces the previous ArbOS-version-specific gate.
