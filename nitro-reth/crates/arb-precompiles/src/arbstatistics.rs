use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbStatistics;

/// ArbStatistics precompile address (0x6f).
pub const ARBSTATISTICS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6f,
]);

const COPY_GAS: u64 = 3;
const SLOAD_GAS: u64 = 800;

pub fn create_arbstatistics_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbstatistics"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbStatistics::ArbStatisticsCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbStatistics::ArbStatisticsCalls;
    let result = match call {
        ArbStatisticsCalls::getStats(_) => handle_get_stats(&input),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_get_stats(input: &PrecompileInput<'_>) -> PrecompileResult {
    // Five Classic-era stats stay zero post-migration; only block number is live.
    let block_number = U256::from(crate::arbsys::get_current_l2_block());
    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&block_number.to_be_bytes::<32>());
    for _ in 0..5 {
        out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    }
    let gas_cost = (SLOAD_GAS + 6 * COPY_GAS).min(input.gas);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
