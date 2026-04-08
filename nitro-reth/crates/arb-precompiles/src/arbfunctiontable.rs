use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbFunctionTable precompile address (0x68).
pub const ARBFUNCTIONTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x68,
]);

const UPLOAD: [u8; 4] = [0xce, 0x2a, 0xe1, 0x59]; // upload(bytes)
const SIZE: [u8; 4] = [0x88, 0x98, 0x70, 0x68]; // size(address)
const GET: [u8; 4] = [0xb4, 0x64, 0x63, 0x1b]; // get(address,uint256)

const COPY_GAS: u64 = 3;

pub fn create_arbfunctiontable_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfunctiontable"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;

    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        UPLOAD => {
            // No-op, returns empty.
            let gas_cost = COPY_GAS.min(gas_limit);
            Ok(PrecompileOutput::new(gas_cost, vec![].into()))
        }
        SIZE => {
            // Returns 0.
            let gas_cost = COPY_GAS.min(gas_limit);
            Ok(PrecompileOutput::new(
                gas_cost,
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET => Err(PrecompileError::other("table is empty")),
        _ => return crate::burn_all_revert(gas_limit),
    };
    crate::gas_check(gas_limit, result)
}
