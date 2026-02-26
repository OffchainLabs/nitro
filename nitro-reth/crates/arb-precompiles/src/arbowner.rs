use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileResult};

/// ArbOwner precompile address (0x70).
pub const ARBOWNER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x70,
]);

pub fn create_arbowner_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbowner"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    // ArbOwner methods are gated by the OwnerPrecompile wrapper.
    // The caller must be verified as a chain owner before any method executes.
    // For now, we reject all calls — owner verification requires checking
    // the chain owners AddressSet, which we'll wire up when the block executor
    // invokes owner-gated precompiles through the proper auth path.
    let _ = &mut input;
    Err(PrecompileError::other(
        "ArbOwner: caller is not a chain owner",
    ))
}
