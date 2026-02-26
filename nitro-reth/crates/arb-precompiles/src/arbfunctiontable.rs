use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbFunctionTable precompile address (0x68).
pub const ARBFUNCTIONTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x68,
]);

const UPLOAD: [u8; 4] = [0x88, 0x3c, 0x9d, 0x6b]; // upload(bytes)
const SIZE: [u8; 4] = [0x17, 0x24, 0x56, 0x7f]; // size(address)
const GET: [u8; 4] = [0xa0, 0x69, 0xee, 0x85]; // get(address,uint256)

const COPY_GAS: u64 = 3;

pub fn create_arbfunctiontable_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfunctiontable"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    let gas = input.gas;

    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        UPLOAD => {
            // No-op, returns empty.
            let gas_cost = COPY_GAS.min(gas);
            Ok(PrecompileOutput::new(gas_cost, vec![].into()))
        }
        SIZE => {
            // Returns 0.
            let gas_cost = COPY_GAS.min(gas);
            Ok(PrecompileOutput::new(
                gas_cost,
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET => Err(PrecompileError::other("table is empty")),
        _ => Err(PrecompileError::other(
            "unknown ArbFunctionTable selector",
        )),
    }
}
