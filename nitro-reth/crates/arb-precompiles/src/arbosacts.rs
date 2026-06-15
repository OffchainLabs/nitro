use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use alloy_sol_types::{SolError, SolInterface};
use arb_context::ArbPrecompileCtx;
use revm::precompile::{PrecompileId, PrecompileResult};
use std::sync::Arc;

use crate::interfaces::IArbosActs;

/// ArbosActs precompile address (0xa4b05).
pub const ARBOSACTS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x0a, 0x4b, 0x05,
]);

pub fn create_arbosacts_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbosacts"), move |input| {
        handler(input, &ctx)
    })
}

/// Every method is invoked by ArbOS internally, never by an EVM caller, so a
/// valid call reverts with `CallerNotArbOS()`. Unknown selectors, static,
/// delegated, or value-bearing calls consume all gas and revert empty.
fn handler(input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let gas_limit = input.gas;
    let known_method = IArbosActs::ArbosActsCalls::abi_decode(input.data).is_ok();
    if !known_method
        || input.is_static
        || input.target_address != input.bytecode_address
        || !input.value.is_zero()
    {
        return crate::burn_all_revert(gas_limit);
    }

    let mut gas_used = 0u64;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());
    crate::revert_sol_error(
        &mut gas_used,
        ctx,
        IArbosActs::CallerNotArbOS {}.abi_encode(),
        gas_limit,
    )
}
