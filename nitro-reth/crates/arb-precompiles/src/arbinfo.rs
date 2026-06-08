use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbInfo, ArbPrecompileError};

/// ArbInfo precompile address (0x65).
pub const ARBINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x65,
]);

const COPY_GAS: u64 = 3;

pub fn create_arbinfo_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbinfo"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());

    let call = match IArbInfo::ArbInfoCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };
    if let Some(r) = crate::reject_nonpayable_value(input.value, input.data, gas_limit, &[]) {
        return r;
    }
    if let Some(r) = crate::reject_delegate_nonpure(
        input.target_address != input.bytecode_address,
        input.data,
        gas_limit,
        &[],
    ) {
        return r;
    }

    use IArbInfo::ArbInfoCalls;
    let result = match call {
        ArbInfoCalls::getBalance(c) => {
            handle_get_balance(&mut input, &mut gas_used, ctx, c.account)
        }
        ArbInfoCalls::getCode(c) => handle_get_code(&mut input, &mut gas_used, ctx, c.account),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn handle_get_balance(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let balance = crate::without_access_list_effect(input.internals_mut(), |internals| {
        internals
            .load_account(addr)
            .map(|acct| acct.data.info.balance)
            .map_err(ArbPrecompileError::fatal)
    })?;
    // BalanceGasEIP1884 (700) is an account read; resultCost (3) is the return copy.
    crate::charge_storage_read(gas_used, ctx, 700);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        balance.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_code(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let code = crate::without_access_list_effect(input.internals_mut(), |internals| {
        internals
            .load_account_code(addr)
            .map(|acct| {
                acct.data
                    .code()
                    .map(|c| c.original_bytes())
                    .unwrap_or_default()
            })
            .map_err(ArbPrecompileError::fatal)
    })?;

    let pad = (32 - code.len() % 32) % 32;
    let mut out = Vec::with_capacity(64 + code.len() + pad);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(code.len()).to_be_bytes::<32>());
    out.extend_from_slice(&code);
    out.extend(std::iter::repeat_n(0u8, pad));

    // ColdSloadCostEIP2929 (2100) is the cold account access; the copy costs
    // (over the code + result) are pure computation.
    let code_words = (code.len() as u64).div_ceil(32);
    let result_words = (out.len() as u64).div_ceil(32);
    crate::charge_storage_read(gas_used, ctx, 2100);
    crate::charge_computation(gas_used, ctx, COPY_GAS * (code_words + result_words));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.into(),
    ))
}
