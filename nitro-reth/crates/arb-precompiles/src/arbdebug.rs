use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Bytes, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use revm::{
    precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult},
    primitives::Log,
};

use crate::{
    interfaces::IArbDebug,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
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
    let input_len = input.data.len();
    let result = match call {
        ArbDebugCalls::becomeChainOwner(_) => handle_become_chain_owner(&mut input),
        ArbDebugCalls::events(c) => handle_events(&mut input, c.flag, c.value),
        ArbDebugCalls::eventsView(_) => handle_events_view(&mut input),
        ArbDebugCalls::customRevert(c) => {
            crate::init_precompile_gas_pure(input_len);
            handle_custom_revert(c.number, gas_limit)
        }
        ArbDebugCalls::legacyError(_) => {
            crate::init_precompile_gas_pure(input_len);
            Err(PrecompileError::other("example legacy error"))
        }
        ArbDebugCalls::panic(_) => panic!("called ArbDebug's debug-only Panic method"),
        ArbDebugCalls::overwriteContractCode(c) => {
            handle_overwrite_contract_code(&mut input, c.target, c.newCode)
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

fn handle_events_view(_input: &mut PrecompileInput<'_>) -> PrecompileResult {
    Err(PrecompileError::other(
        "cannot emit logs in a view method",
    ))
}

fn handle_custom_revert(number: u64, gas_limit: u64) -> PrecompileResult {
    let payload = IArbDebug::Custom {
        _0: number,
        _1: "This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\".to_string(),
        _2: true,
    }
    .abi_encode();
    crate::sol_error_revert(payload, gas_limit)
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

/// `ArbDebug.overwriteContractCode(address target, bytes newCode) -> bytes oldCode`.
/// Replaces the target account's runtime code with `newCode` and returns the
/// previous code, without any code-size or EIP-3541 checks (debug-only).
fn handle_overwrite_contract_code(
    input: &mut PrecompileInput<'_>,
    target: Address,
    new_code: Bytes,
) -> PrecompileResult {
    let gas_limit = input.gas;

    let old_code: Vec<u8> = match input.internals_mut().load_account_code(target) {
        Ok(state_load) => state_load
            .data
            .code()
            .map(|bc| bc.original_byte_slice().to_vec())
            .unwrap_or_default(),
        Err(e) => return Err(PrecompileError::other(format!("load_account_code: {e:?}"))),
    };

    let bytecode = revm::bytecode::Bytecode::new_raw(new_code.clone());
    if let Err(e) = input.internals_mut().set_code(target, bytecode) {
        return Err(PrecompileError::other(format!("set_code: {e:?}")));
    }

    // ABI-encode `bytes memory oldCode`: offset(0x20) | length(N) | data padded.
    let len = old_code.len();
    let padded_len = (len + 31) / 32 * 32;
    let mut out = Vec::with_capacity(64 + padded_len);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(len as u64).to_be_bytes::<32>());
    out.extend_from_slice(&old_code);
    out.resize(64 + padded_len, 0);

    let result_words = (out.len() as u64).div_ceil(32);
    crate::charge_precompile_gas(COPY_GAS.saturating_mul(result_words));
    let gas_used = crate::get_precompile_gas().min(gas_limit);
    Ok(PrecompileOutput::new(gas_used, out.into()))
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
