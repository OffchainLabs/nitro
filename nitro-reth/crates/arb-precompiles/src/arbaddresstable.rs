use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Bytes, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::{address_table::AddressTableError, arbos_state::arbos_from_input, burn::SystemBurner};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbAddressTable, ArbPrecompileError};

/// ArbAddressTable precompile address (0x66).
pub const ARBADDRESSTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x66,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbaddresstable_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaddresstable"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbAddressTable::ArbAddressTableCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbAddressTable::ArbAddressTableCalls as Calls;
    let result = match call {
        Calls::size(_) => handle_size(&mut input),
        Calls::addressExists(c) => handle_address_exists(&mut input, c.addr),
        Calls::lookup(c) => handle_lookup(&mut input, &mut gas_used, c.addr),
        Calls::lookupIndex(c) => handle_lookup_index(&mut input, &mut gas_used, c.index),
        Calls::register(c) => handle_register(&mut input, &mut gas_used, c.addr),
        Calls::compress(c) => handle_compress(&mut input, c.addr),
        Calls::decompress(c) => handle_decompress(&mut input, &mut gas_used, &c.buf, c.offset),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn handle_size(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let size = arb_state
        .address_table
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        U256::from(size).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_address_exists(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let exists = arb_state
        .address_table
        .address_exists(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let value = if exists { U256::from(1u64) } else { U256::ZERO };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_lookup(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let (index, exists) = arb_state
        .address_table
        .lookup(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    if !exists {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_lookup_index(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    index_u256: U256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let index: u64 = index_u256
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .address_table
        .lookup_index(internals, index)
        .map_err(ArbPrecompileError::fatal)?
        .ok_or_else(|| ArbPrecompileError::empty_revert(*gas_used))?;

    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(addr.as_slice());

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_register(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    let (index, already_registered) = arb_state
        .address_table
        .register(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;

    if already_registered {
        return Ok(PrecompileOutput::new(
            (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
            U256::from(index).to_be_bytes::<32>().to_vec().into(),
        ));
    }

    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS);

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_compress(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let rlp_bytes = arb_state
        .address_table
        .compress(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;

    let mut output = Vec::with_capacity(96);
    output.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    output.extend_from_slice(&U256::from(rlp_bytes.len() as u64).to_be_bytes::<32>());
    output.extend_from_slice(&rlp_bytes);
    let pad = (32 - rlp_bytes.len() % 32) % 32;
    output.extend(std::iter::repeat_n(0u8, pad));

    let result_words = (output.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS + result_words * COPY_GAS).min(gas_limit),
        output.into(),
    ))
}

fn handle_decompress(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    buf: &Bytes,
    offset: U256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let data_len = input.data.len();
    let ioffset: usize = offset
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    if ioffset >= buf.len() {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    let slice = &buf[ioffset..];

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    let (addr, bytes_read, raw_address) = arb_state
        .address_table
        .decompress(internals, slice)
        .map_err(|e| match e {
            AddressTableError::Storage(s) => ArbPrecompileError::fatal(s),
            AddressTableError::InvalidEncoding | AddressTableError::IndexOutOfRange(_) => {
                ArbPrecompileError::empty_revert(*gas_used)
            }
        })?;

    let mut output = Vec::with_capacity(64);
    output.extend_from_slice(&alloy_primitives::B256::left_padding_from(addr.as_slice()).0);
    output.extend_from_slice(&U256::from(bytes_read).to_be_bytes::<32>());

    let body_sloads: u64 = if raw_address { 1 } else { 3 };
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    Ok(PrecompileOutput::new(
        (body_sloads * SLOAD_GAS + (arg_words + 2) * COPY_GAS).min(gas_limit),
        output.into(),
    ))
}
