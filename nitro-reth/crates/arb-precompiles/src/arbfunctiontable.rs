use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbFunctionTable;

/// ArbFunctionTable precompile address (0x68).
pub const ARBFUNCTIONTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x68,
]);

const COPY_GAS: u64 = 3;

pub fn create_arbfunctiontable_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfunctiontable"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbFunctionTable::ArbFunctionTableCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbFunctionTable::ArbFunctionTableCalls;
    let result = match call {
        ArbFunctionTableCalls::upload(_) => Ok(PrecompileOutput::new(
            COPY_GAS.min(gas_limit),
            vec![].into(),
        )),
        ArbFunctionTableCalls::size(_) => Ok(PrecompileOutput::new(
            COPY_GAS.min(gas_limit),
            U256::ZERO.to_be_bytes::<32>().to_vec().into(),
        )),
        ArbFunctionTableCalls::get(_) => Err(PrecompileError::other("table is empty")),
    };
    crate::gas_check(gas_limit, result)
}
