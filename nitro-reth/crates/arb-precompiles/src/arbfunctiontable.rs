use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbFunctionTable, ArbPrecompileError};

/// ArbFunctionTable precompile address (0x68).
pub const ARBFUNCTIONTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x68,
]);

const COPY_GAS: u64 = 3;

pub fn create_arbfunctiontable_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfunctiontable"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbFunctionTable::ArbFunctionTableCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbFunctionTable::ArbFunctionTableCalls;
    let result = match call {
        // Upload: no-op. Cost = OpenArbosState + argsCost (pre-charged).
        ArbFunctionTableCalls::upload(_) => Ok(PrecompileOutput::new(
            gas_used.min(gas_limit),
            vec![].into(),
        )),
        // Size: no-op returning 0. Cost = OpenArbosState + argsCost + 1-word resultCost.
        ArbFunctionTableCalls::size(_) => {
            crate::charge_precompile_gas(&mut gas_used, COPY_GAS);
            Ok(PrecompileOutput::new(
                gas_used.min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        // Get unconditionally reverts (table is empty). gas_check will return
        // accumulated_gas (OpenArbosState + argsCost) on the revert path.
        ArbFunctionTableCalls::get(_) => Err(ArbPrecompileError::empty_revert(gas_used).into()),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}
