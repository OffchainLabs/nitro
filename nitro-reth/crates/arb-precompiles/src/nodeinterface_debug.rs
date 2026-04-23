use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::INodeInterfaceDebug;

/// NodeInterfaceDebug virtual contract address (0xc9).
pub const NODE_INTERFACE_DEBUG_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xc9,
]);

const COPY_GAS: u64 = 3;

pub fn create_nodeinterface_debug_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("nodeinterfacedebug"), handler)
}

fn handler(input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match INodeInterfaceDebug::NodeInterfaceDebugCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use INodeInterfaceDebug::NodeInterfaceDebugCalls;
    let result = match call {
        NodeInterfaceDebugCalls::getRetryable(_) => handle_get_retryable(&input),
    };
    crate::gas_check(gas_limit, result)
}

/// Returns a well-formed empty `RetryableInfo` — bridge tooling gets a valid
/// ABI response; populating it requires RPC-layer state access.
fn handle_get_retryable(input: &PrecompileInput<'_>) -> PrecompileResult {
    let mut out = vec![0u8; 7 * 32 + 32];
    U256::from(7u64 * 32)
        .to_be_bytes::<32>()
        .iter()
        .enumerate()
        .for_each(|(i, b)| out[6 * 32 + i] = *b);
    Ok(PrecompileOutput::new(COPY_GAS.min(input.gas), out.into()))
}
