use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use arb_context::ArbPrecompileCtx;
use revm::precompile::{PrecompileId, PrecompileResult};
use std::sync::Arc;

use crate::ArbPrecompileError;

/// ArbosActs precompile address (0xa4b05).
pub const ARBOSACTS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x0a, 0x4b, 0x05,
]);

pub fn create_arbosacts_precompile(_ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbosacts"), handler)
}

fn handler(_input: PrecompileInput<'_>) -> PrecompileResult {
    Err(ArbPrecompileError::empty_revert(0).into())
}
