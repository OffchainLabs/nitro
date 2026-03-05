use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbStatistics precompile address (0x6f).
pub const ARBSTATISTICS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6f,
]);

const GET_STATS: [u8; 4] = [0xe1, 0x1b, 0x84, 0xd8]; // getStats()

const COPY_GAS: u64 = 3;
const SLOAD_GAS: u64 = 800;

pub fn create_arbstatistics_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbstatistics"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        GET_STATS => handle_get_stats(&input),
        _ => Err(PrecompileError::other("unknown ArbStatistics selector")),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_get_stats(input: &PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;

    // Returns (blockNumber, 0, 0, 0, 0, 0).
    // The five Classic-era stats are also zero in Nitro (never populated post-migration).
    // Use the L2 block number (block_env.number holds L1 in Arbitrum).
    let block_number = U256::from(crate::arbsys::get_current_l2_block());
    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&block_number.to_be_bytes::<32>());
    for _ in 0..5 {
        out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    }

    // Go: OAS(800) + 0 body + resultCost = 6 words × 3 = 18.
    let gas_cost = (SLOAD_GAS + 6 * COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
