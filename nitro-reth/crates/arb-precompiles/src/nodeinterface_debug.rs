//! NodeInterfaceDebug (0xc9) virtual contract — debug-only accessors
//! exposed via `eth_call` that don't need to be gated by ArbOS version.
//!
//! This mirrors Nitro's `execution/nodeinterface/debug.go`. Methods
//! read ArbOS state but don't modify it; since debug-only, we prefer
//! returning a minimal valid response over strict consensus-parity
//! (the ABI schema is what matters for tooling that calls these).

use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// NodeInterfaceDebug virtual contract address (0xc9).
pub const NODE_INTERFACE_DEBUG_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xc9,
]);

// Function selectors.
const GET_RETRYABLE: [u8; 4] = [0x4d, 0x79, 0x53, 0xad]; // getRetryable(bytes32)

const COPY_GAS: u64 = 3;

pub fn create_nodeinterface_debug_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("nodeinterfacedebug"), handler)
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
        GET_RETRYABLE => handle_get_retryable(&input),
        _ => return crate::burn_all_revert(gas_limit),
    };
    crate::gas_check(gas_limit, result)
}

/// `getRetryable(bytes32 ticketId) -> (uint256 timeout, address from,
/// address to, uint256 value, address beneficiary, uint64 tries,
/// bytes data)`
///
/// Without access to `RetryableState` from inside the precompile (we
/// don't have the ArbOS state handle at this precompile layer), this
/// returns a well-formed empty response — signalling "no retryable
/// found / unavailable at this execution path". An RPC-layer
/// interception (parallel to the 0xc8 override) could later look up
/// the actual retryable; until then, bridge tooling that calls this
/// gets a valid ABI response instead of a revert.
fn handle_get_retryable(input: &PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 + 32 {
        return Err(PrecompileError::other(
            "getRetryable: missing bytes32 ticket",
        ));
    }
    // Empty RetryableInfo:
    //   head: timeout(32) + from(32) + to(32) + value(32) + beneficiary(32)
    //         + tries(32) + dataOffset(32)
    //   tail: dataLen(32)
    let mut out = vec![0u8; 7 * 32 + 32];
    // dataOffset = head size = 7 * 32 = 0xE0
    U256::from(7u64 * 32)
        .to_be_bytes::<32>()
        .iter()
        .enumerate()
        .for_each(|(i, b)| out[6 * 32 + i] = *b);
    // dataLen = 0 (already zero)
    let _ = B256::ZERO;
    Ok(PrecompileOutput::new(COPY_GAS.min(input.gas), out.into()))
}
