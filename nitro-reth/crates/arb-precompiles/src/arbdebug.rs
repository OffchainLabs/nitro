use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileResult};

/// ArbDebug precompile address (0xff).
pub const ARBDEBUG_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xff,
]);

pub fn create_arbdebug_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbdebug"), handler)
}

fn handler(_input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbDebug is gated by the DebugPrecompile wrapper in Go.
    // In production, all calls are rejected.
    Err(PrecompileError::other(
        "ArbDebug is only available in debug mode",
    ))
}
