use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbRetryableTx precompile address (0x6e).
pub const ARBRETRYABLETX_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6e,
]);

// Function selectors.
const REDEEM: [u8; 4] = [0xed, 0xa1, 0x12, 0x2c];
const GET_LIFETIME: [u8; 4] = [0x81, 0xe6, 0xe0, 0x83];
const GET_TIMEOUT: [u8; 4] = [0x9f, 0x10, 0x25, 0xc6];
const KEEPALIVE: [u8; 4] = [0xf0, 0xb2, 0x1a, 0x41];
const GET_BENEFICIARY: [u8; 4] = [0xba, 0x20, 0xdd, 0xa4];
const CANCEL: [u8; 4] = [0xc4, 0xd2, 0x52, 0xf5];
const GET_CURRENT_REDEEMER: [u8; 4] = [0xde, 0x4b, 0xa2, 0xb3];
const SUBMIT_RETRYABLE: [u8; 4] = [0xc9, 0xf9, 0x5d, 0x32];

/// Default retryable lifetime: 7 days in seconds.
const RETRYABLE_LIFETIME_SECONDS: u64 = 7 * 24 * 60 * 60;

const COPY_GAS: u64 = 3;

pub fn create_arbretryabletx_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbretryabletx"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_limit = input.gas;

    match selector {
        GET_LIFETIME => {
            // Returns the default lifetime (7 days).
            let lifetime = U256::from(RETRYABLE_LIFETIME_SECONDS);
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                lifetime.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_CURRENT_REDEEMER => {
            // Returns zero address when not in a retryable redeem context.
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SUBMIT_RETRYABLE => {
            // This method is not callable — it exists only for ABI/explorer purposes.
            Err(PrecompileError::other("not callable"))
        }
        REDEEM | GET_TIMEOUT | KEEPALIVE | GET_BENEFICIARY | CANCEL => {
            // These methods require retryable state access.
            Err(PrecompileError::other(
                "retryable operations require state access",
            ))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbRetryableTx selector",
        )),
    }
}
