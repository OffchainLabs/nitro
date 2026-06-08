use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::interfaces::IArbStatistics;

/// ArbStatistics precompile address (0x6f).
pub const ARBSTATISTICS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6f,
]);

const COPY_GAS: u64 = 3;

pub fn create_arbstatistics_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbstatistics"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());

    let call = match IArbStatistics::ArbStatisticsCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };
    if let Some(r) = crate::reject_nonpayable_value(input.value, input.data, gas_limit, &[]) {
        return r;
    }
    if let Some(r) = crate::reject_delegate_nonpure(
        input.target_address != input.bytecode_address,
        input.data,
        gas_limit,
        &[],
    ) {
        return r;
    }

    use IArbStatistics::ArbStatisticsCalls;
    let result = match call {
        ArbStatisticsCalls::getStats(_) => handle_get_stats(&input, &mut gas_used, ctx),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn handle_get_stats(
    input: &PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    // Five Classic-era stats stay zero post-migration; only block number is live.
    let block_number = U256::from(ctx.block.l2_block_number);
    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&block_number.to_be_bytes::<32>());
    for _ in 0..5 {
        out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    }
    crate::charge_computation(gas_used, ctx, 6 * COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(input.gas),
        out.into(),
    ))
}
