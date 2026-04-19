use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use revm::{
    precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult},
    primitives::Log,
};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
    ROOT_STORAGE_KEY,
};

/// ArbDebug precompile address (0xff).
pub const ARBDEBUG_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xff,
]);

const BECOME_CHAIN_OWNER: [u8; 4] = [0x0e, 0x5b, 0xbc, 0x11];
const EVENTS: [u8; 4] = [0x7b, 0x99, 0x63, 0xef];
const EVENTS_VIEW: [u8; 4] = [0x8e, 0x5f, 0x30, 0xab];
const CUSTOM_REVERT: [u8; 4] = [0x7e, 0xa8, 0x9f, 0x8b];
const LEGACY_ERROR: [u8; 4] = [0x1e, 0x48, 0xfe, 0x82];
const PANIC: [u8; 4] = [0x47, 0x00, 0xd3, 0x05];

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;

pub fn create_arbdebug_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbdebug"), handler)
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
        BECOME_CHAIN_OWNER => handle_become_chain_owner(&mut input),
        EVENTS => handle_events(&mut input),
        EVENTS_VIEW => handle_events_view(&mut input),
        CUSTOM_REVERT => handle_custom_revert(&input),
        LEGACY_ERROR => Err(PrecompileError::other("example legacy error")),
        PANIC => panic!("called ArbDebug's debug-only Panic method"),
        _ => return crate::burn_all_revert(gas_limit),
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
    if existing == U256::ZERO {
        let size_slot = map_slot(set_key.as_slice(), 0);
        let size = sload(input, size_slot)?;
        let new_size = u64::try_from(size).unwrap_or(0) + 1;

        let new_pos_slot = map_slot(set_key.as_slice(), new_size);
        sstore(input, new_pos_slot, U256::from_be_slice(caller.as_slice()))?;
        sstore(input, member_slot, U256::from(new_size))?;
        sstore(input, size_slot, U256::from(new_size))?;
    }

    let gas_used = 2 * SLOAD_GAS + 3 * SSTORE_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_events(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    let flag = data[35] != 0;
    let value = B256::from_slice(&data[36..68]);
    let caller = input.caller;
    let value_received = input.value;

    emit_basic_event(input, !flag, value);
    emit_mixed_event(input, flag, !flag, value, ARBDEBUG_ADDRESS, caller);

    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(B256::left_padding_from(caller.as_slice()).as_slice());
    out.extend_from_slice(&value_received.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(gas_limit.min(3000), out.into()))
}

fn handle_events_view(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    if input.is_static {
        return Err(PrecompileError::other(
            "cannot emit logs in static call context",
        ));
    }
    let zero_value = B256::ZERO;
    emit_basic_event(input, false, zero_value);
    emit_mixed_event(input, true, false, zero_value, ARBDEBUG_ADDRESS, input.caller);
    Ok(PrecompileOutput::new(input.gas.min(3000), Vec::new().into()))
}

fn handle_custom_revert(input: &PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let number = U256::from_be_slice(&data[4..36]);
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

fn sstore(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    Ok(())
}

fn emit_basic_event(input: &mut PrecompileInput<'_>, flag: bool, value: B256) {
    let topic0 = keccak256("Basic(bool,bytes32)");
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
    let topic0 = keccak256("Mixed(bool,bool,bytes32,address,address)");
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
