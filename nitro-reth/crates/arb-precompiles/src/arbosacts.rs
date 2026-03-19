use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileResult};

/// ArbosActs precompile address (0xa4b05).
pub const ARBOSACTS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x0a, 0x4b, 0x05,
]);

pub fn create_arbosacts_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbosacts"), handler)
}

fn handler(_input: PrecompileInput<'_>) -> PrecompileResult {
    // All ArbosActs methods reject non-ArbOS callers.
    // These are internal lifecycle hooks invoked by the block production engine,
    // never by external transactions.
    Err(PrecompileError::other("caller is not ArbOS"))
}
