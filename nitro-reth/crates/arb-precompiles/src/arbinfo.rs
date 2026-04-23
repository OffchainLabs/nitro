use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbInfo;

/// ArbInfo precompile address (0x65).
pub const ARBINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x65,
]);

const COPY_GAS: u64 = 3;
const SLOAD_GAS: u64 = 800;

pub fn create_arbinfo_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbinfo"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbInfo::ArbInfoCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbInfo::ArbInfoCalls;
    let result = match call {
        ArbInfoCalls::getBalance(c) => handle_get_balance(&mut input, c.account),
        ArbInfoCalls::getCode(c) => handle_get_code(&mut input, c.account),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_get_balance(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    let acct = internals
        .load_account(addr)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    let balance = acct.data.info.balance;
    // OpenArbosState (800) + argsCost (3) + BalanceGasEIP1884 (700) + resultCost (3).
    let gas_cost = (SLOAD_GAS + 3 + 700 + COPY_GAS).min(gas_limit);

    Ok(PrecompileOutput::new(
        gas_cost,
        balance.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_code(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    let acct = internals
        .load_account_code(addr)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    let code = acct
        .data
        .code()
        .map(|c| c.original_bytes())
        .unwrap_or_default();

    let pad = (32 - code.len() % 32) % 32;
    let mut out = Vec::with_capacity(64 + code.len() + pad);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(code.len()).to_be_bytes::<32>());
    out.extend_from_slice(&code);
    out.extend(std::iter::repeat_n(0u8, pad));

    // OpenArbosState (800) + argsCost (3) + ColdSloadCostEIP2929 (2100) +
    // copy * words(code) + copy * words(result).
    let code_words = (code.len() as u64).div_ceil(32);
    let result_words = (out.len() as u64).div_ceil(32);
    let gas_cost =
        (SLOAD_GAS + 3 + 2100 + COPY_GAS * code_words + COPY_GAS * result_words).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
