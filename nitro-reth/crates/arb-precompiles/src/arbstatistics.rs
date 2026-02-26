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

pub fn create_arbstatistics_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbstatistics"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        GET_STATS => handle_get_stats(&input),
        _ => Err(PrecompileError::other("unknown ArbStatistics selector")),
    }
}

fn handle_get_stats(input: &PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;

    // Returns (blockNumber, 0, 0, 0, 0, 0) — classic stats are stubbed.
    // block_number is not directly available on PrecompileInput,
    // so we return zero for all fields as a safe default.
    let block_number = U256::ZERO;
    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&block_number.to_be_bytes::<32>());
    for _ in 0..5 {
        out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    }

    let gas_cost = COPY_GAS.min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
