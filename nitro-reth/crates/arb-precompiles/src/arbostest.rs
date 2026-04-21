use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbosTest precompile address (0x69). Burns arbitrary amounts of L2 gas.
pub const ARBOSTEST_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x69,
]);

const BURN_ARB_GAS: [u8; 4] = [0xbb, 0x34, 0x80, 0xf9]; // burnArbGas(uint256)

pub fn create_arbostest_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbostest"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if !crate::allow_debug_precompiles() {
        return crate::burn_all_revert(gas_limit);
    }
    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }
    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        BURN_ARB_GAS => handle_burn_arb_gas(&input),
        _ => return crate::burn_all_revert(gas_limit),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_burn_arb_gas(input: &PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(gas_limit);
    }
    let amount = U256::from_be_slice(&data[4..36]);
    let to_burn: u64 = amount.try_into().unwrap_or(u64::MAX);
    let charge = to_burn.min(gas_limit);
    Ok(PrecompileOutput::new(charge, Vec::new().into()))
}
