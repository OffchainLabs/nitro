use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileResult};

/// ArbNativeTokenManager precompile address (0x75).
pub const ARBNATIVETOKENMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x75,
]);

// Function selectors.
const MINT_NATIVE_TOKEN: [u8; 4] = [0xf2, 0xe2, 0x34, 0x70];
const BURN_NATIVE_TOKEN: [u8; 4] = [0xa7, 0x54, 0x40, 0x2b];

pub fn create_arbnativetokenmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbnativetokenmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        MINT_NATIVE_TOKEN | BURN_NATIVE_TOKEN => {
            // Requires native token owner access.
            let _ = &mut input;
            Err(PrecompileError::other(
                "caller is not a native token owner",
            ))
        }
        _ => Err(PrecompileError::other("unknown selector")),
    }
}
