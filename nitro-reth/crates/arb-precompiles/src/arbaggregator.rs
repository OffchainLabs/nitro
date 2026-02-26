use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbAggregator precompile address (0x6d).
pub const ARBAGGREGATOR_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6d,
]);

/// Default batch poster address (the sequencer).
const BATCH_POSTER_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75,
    0x65, 0x6e, 0x63, 0x65, 0x72,
]);

// Function selectors.
const GET_PREFERRED_AGGREGATOR: [u8; 4] = [0x52, 0xf1, 0x07, 0x40];
const GET_DEFAULT_AGGREGATOR: [u8; 4] = [0x87, 0x58, 0x83, 0xf2];
const GET_BATCH_POSTERS: [u8; 4] = [0xe1, 0x05, 0x73, 0xa3];
const ADD_BATCH_POSTER: [u8; 4] = [0xdf, 0x41, 0xe1, 0xe2];
const GET_FEE_COLLECTOR: [u8; 4] = [0x9c, 0x2c, 0x5b, 0xb5];
const SET_FEE_COLLECTOR: [u8; 4] = [0x29, 0x14, 0x97, 0x99];
const GET_TX_BASE_FEE: [u8; 4] = [0x04, 0x97, 0x64, 0xaf];
const SET_TX_BASE_FEE: [u8; 4] = [0x5b, 0xe6, 0x88, 0x8b];

const COPY_GAS: u64 = 3;

pub fn create_arbaggregator_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaggregator"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_limit = input.gas;

    match selector {
        GET_PREFERRED_AGGREGATOR => {
            // Deprecated: always returns (BatchPosterAddress, true).
            let mut out = Vec::with_capacity(96);
            // ABI offset for the tuple
            out.extend_from_slice(&U256::from(0x40u64).to_be_bytes::<32>());
            // isDefault = true
            out.extend_from_slice(&U256::from(1u64).to_be_bytes::<32>());
            // address (left-padded)
            let mut addr_word = [0u8; 32];
            addr_word[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            out.extend_from_slice(&addr_word);
            Ok(PrecompileOutput::new(COPY_GAS.min(gas_limit), out.into()))
        }
        GET_DEFAULT_AGGREGATOR => {
            // Deprecated: always returns BatchPosterAddress.
            let mut out = [0u8; 32];
            out[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                out.to_vec().into(),
            ))
        }
        GET_TX_BASE_FEE => {
            // Deprecated: always returns 0.
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SET_TX_BASE_FEE => {
            // Deprecated: no-op.
            Ok(PrecompileOutput::new(COPY_GAS.min(gas_limit), vec![].into()))
        }
        GET_BATCH_POSTERS | ADD_BATCH_POSTER | GET_FEE_COLLECTOR | SET_FEE_COLLECTOR => {
            // These methods require batch poster table state access.
            Err(PrecompileError::other(
                "batch poster table operations not yet supported",
            ))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbAggregator selector",
        )),
    }
}
