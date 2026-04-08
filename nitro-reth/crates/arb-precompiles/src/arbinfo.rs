use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbInfo precompile address (0x65).
pub const ARBINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x65,
]);

// Function selectors.
const GET_BALANCE: [u8; 4] = [0xf8, 0xb2, 0xcb, 0x4f]; // getBalance(address)
const GET_CODE: [u8; 4] = [0x7e, 0x10, 0x5c, 0xe2]; // getCode(address)

const COPY_GAS: u64 = 3;
const SLOAD_GAS: u64 = 800;

pub fn create_arbinfo_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbinfo"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        GET_BALANCE => handle_get_balance(&mut input),
        GET_CODE => handle_get_code(&mut input),
        _ => return crate::burn_all_revert(gas_limit),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_get_balance(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let addr = Address::from_slice(&data[16..36]);
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

fn handle_get_code(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let addr = Address::from_slice(&data[16..36]);
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

    // ABI-encode: offset(32) + length(32) + data (padded)
    let pad = (32 - code.len() % 32) % 32;
    let mut out = Vec::with_capacity(64 + code.len() + pad);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(code.len()).to_be_bytes::<32>());
    out.extend_from_slice(&code);
    out.extend(std::iter::repeat_n(0u8, pad));

    // OpenArbosState (800) + argsCost (3) + ColdSloadCostEIP2929 (2100) +
    // CopyGas * WordsForBytes(code_len) + resultCost for ABI-encoded bytes.
    let code_words = (code.len() as u64).div_ceil(32);
    let result_words = (out.len() as u64).div_ceil(32);
    let gas_cost =
        (SLOAD_GAS + 3 + 2100 + COPY_GAS * code_words + COPY_GAS * result_words).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
