use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Bytes, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use revm::{
    precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult},
    primitives::Log,
};

use crate::interfaces::IArbDebug;
use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
    ROOT_STORAGE_KEY,
};

/// ArbDebug precompile address (0xff).
pub const ARBDEBUG_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xff,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;
const LOG_GAS: u64 = 375;
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8;

pub fn create_arbdebug_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbdebug"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    if !crate::allow_debug_precompiles() {
        return crate::burn_all_revert(gas_limit);
    }
    crate::init_precompile_gas(input.data.len());

    let call = match IArbDebug::ArbDebugCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbDebug::ArbDebugCalls;
    let result = match call {
        ArbDebugCalls::becomeChainOwner(_) => handle_become_chain_owner(&mut input),
        ArbDebugCalls::events(c) => handle_events(&mut input, c.flag, c.value),
        ArbDebugCalls::eventsView(_) => handle_events_view(&mut input),
        ArbDebugCalls::customRevert(c) => handle_custom_revert(c.number),
        ArbDebugCalls::legacyError(_) => Err(PrecompileError::other("example legacy error")),
        ArbDebugCalls::panic(_) => panic!("called ArbDebug's debug-only Panic method"),
        ArbDebugCalls::overwriteContractCode(_) => {
            Err(PrecompileError::other("overwriteContractCode not implemented"))
        }
    };

    crate::gas_check(gas_limit, result)
}

fn handle_become_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let caller = input.caller;
    let gas_limit = input.gas;

    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(caller.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);

    let existing = sload(input, member_slot)?;
    let gas_used = if existing == U256::ZERO {
        let size_slot = map_slot(set_key.as_slice(), 0);
        let size = sload(input, size_slot)?;
        let new_size = u64::try_from(size).unwrap_or(0) + 1;

        let new_pos_slot = map_slot(set_key.as_slice(), new_size);
        sstore(input, new_pos_slot, U256::from_be_slice(caller.as_slice()))?;
        sstore(input, member_slot, U256::from(new_size))?;
        sstore(input, size_slot, U256::from(new_size))?;

        4 * SLOAD_GAS + 3 * SSTORE_GAS
    } else {
        2 * SLOAD_GAS
    };

    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_events(input: &mut PrecompileInput<'_>, flag: bool, value: B256) -> PrecompileResult {
    let gas_limit = input.gas;
    let data_len = input.data.len();
    let caller = input.caller;
    let value_received = input.value;

    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    emit_basic_event(input, !flag, value);
    emit_mixed_event(input, flag, !flag, value, ARBDEBUG_ADDRESS, caller);

    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(B256::left_padding_from(caller.as_slice()).as_slice());
    out.extend_from_slice(&value_received.to_be_bytes::<32>());

    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    let result_words = (out.len() as u64).div_ceil(32);
    let basic_log_gas = LOG_GAS + LOG_TOPIC_GAS * 2 + LOG_DATA_GAS * 32;
    let mixed_log_gas = LOG_GAS + LOG_TOPIC_GAS * 4 + LOG_DATA_GAS * 64;
    let gas_cost =
        SLOAD_GAS + COPY_GAS * arg_words + basic_log_gas + mixed_log_gas + COPY_GAS * result_words;

    Ok(PrecompileOutput::new(gas_cost.min(gas_limit), out.into()))
}

fn handle_events_view(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    if input.is_static {
        return Err(PrecompileError::other(
            "cannot emit logs in static call context",
        ));
    }
    let zero_value = B256::ZERO;
    emit_basic_event(input, false, zero_value);
    emit_mixed_event(
        input,
        true,
        false,
        zero_value,
        ARBDEBUG_ADDRESS,
        input.caller,
    );
    Ok(PrecompileOutput::new(
        input.gas.min(3000),
        Vec::new().into(),
    ))
}

fn handle_custom_revert(number: u64) -> PrecompileResult {
    Err(PrecompileError::other(format!(
        "custom error {number}: This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\"
    )))
}

fn sload(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let v = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    crate::charge_precompile_gas(SLOAD_GAS);
    Ok(v.data)
}

fn sstore(input: &mut PrecompileInput<'_>, slot: U256, value: U256) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    Ok(())
}

fn emit_basic_event(input: &mut PrecompileInput<'_>, flag: bool, value: B256) {
    let topic0 = IArbDebug::Basic::SIGNATURE_HASH;
    let topic1 = value;
    let mut data = [0u8; 32];
    if flag {
        data[31] = 1;
    }
    input.internals_mut().log(Log::new_unchecked(
        ARBDEBUG_ADDRESS,
        vec![topic0, topic1],
        Bytes::copy_from_slice(&data),
    ));
}

fn emit_mixed_event(
    input: &mut PrecompileInput<'_>,
    flag1: bool,
    flag2: bool,
    value: B256,
    addr1: Address,
    addr2: Address,
) {
    let topic0 = IArbDebug::Mixed::SIGNATURE_HASH;
    let mut t1 = [0u8; 32];
    if flag1 {
        t1[31] = 1;
    }
    let topic1 = B256::from(t1);
    let topic2 = value;
    let topic3 = B256::left_padding_from(addr2.as_slice());
    let mut data = Vec::with_capacity(64);
    let mut flag2_word = [0u8; 32];
    if flag2 {
        flag2_word[31] = 1;
    }
    data.extend_from_slice(&flag2_word);
    data.extend_from_slice(B256::left_padding_from(addr1.as_slice()).as_slice());
    input.internals_mut().log(Log::new_unchecked(
        ARBDEBUG_ADDRESS,
        vec![topic0, topic1, topic2, topic3],
        Bytes::copy_from_slice(&data),
    ));
}
