use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbWasm precompile address (0x71).
pub const ARBWASM_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x71,
]);

// Function selectors — view methods returning Stylus program config.
const STYLUS_VERSION: [u8; 4] = [0xf2, 0x8a, 0x04, 0x99];
const INK_PRICE: [u8; 4] = [0xeb, 0xf5, 0xd2, 0x51];
const MAX_STACK_DEPTH: [u8; 4] = [0x19, 0x4a, 0xa2, 0x8e];
const FREE_PAGES: [u8; 4] = [0xb6, 0x9d, 0xb8, 0x5e];
const PAGE_GAS: [u8; 4] = [0x96, 0x76, 0xa4, 0x67];
const PAGE_RAMP: [u8; 4] = [0x56, 0xc1, 0x80, 0x1c];
const PAGE_LIMIT: [u8; 4] = [0x20, 0xf0, 0x02, 0xea];
const MIN_INIT_GAS: [u8; 4] = [0x5b, 0x19, 0x32, 0x87];
const INIT_COST_SCALAR: [u8; 4] = [0x67, 0x46, 0x27, 0x93];
const EXPIRY_DAYS: [u8; 4] = [0xee, 0xe2, 0x2a, 0xa3];
const KEEPALIVE_DAYS: [u8; 4] = [0xe7, 0xfb, 0x85, 0x75];
const BLOCK_CACHE_SIZE: [u8; 4] = [0xd2, 0xfb, 0xa3, 0xc5];
const ACTIVATE_PROGRAM: [u8; 4] = [0x72, 0x93, 0x80, 0x88];
const CODEHASH_KEEPALIVE: [u8; 4] = [0xe7, 0xf6, 0x2c, 0x15];
const CODEHASH_VERSION: [u8; 4] = [0xb4, 0xb7, 0xc5, 0xf5];
const CODEHASH_ASM_SIZE: [u8; 4] = [0x5f, 0xd3, 0x5d, 0xea];
const PROGRAM_VERSION: [u8; 4] = [0x70, 0x46, 0x7c, 0x7c];
const PROGRAM_INIT_GAS: [u8; 4] = [0x8e, 0x15, 0xc4, 0x17];
const PROGRAM_MEMORY_FOOTPRINT: [u8; 4] = [0x95, 0x48, 0xea, 0xb0];
const PROGRAM_TIME_LEFT: [u8; 4] = [0x63, 0x5b, 0x36, 0x42];

const COPY_GAS: u64 = 3;

pub fn create_arbwasm_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasm"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbWasm requires ArbOS >= 30 (Stylus).
    if let Some(result) = crate::check_precompile_version(
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
    ) {
        return result;
    }

    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_cost = COPY_GAS.min(input.gas);

    match selector {
        // View methods returning uint values — return defaults.
        STYLUS_VERSION => ok_u256(gas_cost, U256::ZERO),
        INK_PRICE | MAX_STACK_DEPTH | FREE_PAGES | PAGE_GAS | PAGE_RAMP | PAGE_LIMIT
        | MIN_INIT_GAS | INIT_COST_SCALAR | EXPIRY_DAYS | KEEPALIVE_DAYS | BLOCK_CACHE_SIZE => {
            ok_u256(gas_cost, U256::ZERO)
        }
        // Program queries — return zero/error.
        CODEHASH_VERSION | CODEHASH_ASM_SIZE | PROGRAM_VERSION | PROGRAM_INIT_GAS
        | PROGRAM_MEMORY_FOOTPRINT | PROGRAM_TIME_LEFT => ok_u256(gas_cost, U256::ZERO),
        // State-modifying.
        ACTIVATE_PROGRAM => {
            let _ = &mut input;
            Err(PrecompileError::other("Stylus not yet supported"))
        }
        CODEHASH_KEEPALIVE => {
            let _ = &mut input;
            Err(PrecompileError::other("Stylus not yet supported"))
        }
        _ => Err(PrecompileError::other("unknown ArbWasm selector")),
    }
}

fn ok_u256(gas_cost: u64, value: U256) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        gas_cost,
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}
