use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbosTest;

/// ArbosTest precompile address (0x69). Burns arbitrary amounts of L2 gas.
pub const ARBOSTEST_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x69,
]);

pub fn create_arbostest_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbostest"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    if !crate::allow_debug_precompiles() {
        return crate::burn_all_revert(gas_limit);
    }
    crate::init_precompile_gas(input.data.len());

    let call = match IArbosTest::ArbosTestCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbosTest::ArbosTestCalls;
    let result = match call {
        ArbosTestCalls::burnArbGas(c) => handle_burn_arb_gas(gas_limit, c.gasAmount),
    };
    crate::gas_check(gas_limit, result)
}

fn handle_burn_arb_gas(gas_limit: u64, amount: U256) -> PrecompileResult {
    // Nitro's `BurnArbGas(c ctx, gasAmount huge)` is pure (no evm.StateDB
    // access), so the framework skips OpenArbosState. Cost = argsCost (1
    // word = 3 gas) + the gasAmount the method explicitly burns.
    const ARGS_COST: u64 = 3;
    let to_burn: u64 = amount.try_into().unwrap_or(u64::MAX);
    Ok(PrecompileOutput::new(
        ARGS_COST.saturating_add(to_burn).min(gas_limit),
        Vec::new().into(),
    ))
}
