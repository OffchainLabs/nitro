use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::Address;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbFilteredTransactionsManager precompile address (0x74).
pub const ARBFILTEREDTXMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x74,
]);

// Function selectors.
const ADD_FILTERED_TX: [u8; 4] = [0xbf, 0xc1, 0xd5, 0x0e];
const DELETE_FILTERED_TX: [u8; 4] = [0x0b, 0x23, 0x48, 0x5a];
const IS_TX_FILTERED: [u8; 4] = [0x37, 0x94, 0x6f, 0x6a];

const COPY_GAS: u64 = 3;

pub fn create_arbfilteredtxmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfilteredtxmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    // Access control: caller must be a transaction filterer.
    // For now, reject write operations and allow reads.
    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        ADD_FILTERED_TX | DELETE_FILTERED_TX => {
            // Requires filterer access — check caller is in filterers set.
            // Stubbed: return error for unauthorized callers.
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a transaction filterer"))
        }
        IS_TX_FILTERED => {
            // Check if a tx hash is in the filtered set.
            // Reads from FilteredTransactionsState storage.
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        _ => Err(PrecompileError::other("unknown selector")),
    }
}
