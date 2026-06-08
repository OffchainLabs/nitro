use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Bytes, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::address_table::AddressTableError;
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
const SSTORE_ZERO_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;

pub fn create_arbaddresstable_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaddresstable"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());

    let call = match IArbAddressTable::ArbAddressTableCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };
    if let Some(r) = crate::reject_nonpayable_value(input.value, input.data, gas_limit, &[]) {
        return r;
    }
    if let Some(r) = crate::reject_static_write(
        input.is_static,
        input.data,
        gas_limit,
        &[[0xf6, 0xa4, 0x55, 0xa2], [0x44, 0x20, 0xe4, 0x86]],
    ) {
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

    use IArbAddressTable::ArbAddressTableCalls as Calls;
    let result = match call {
        Calls::size(_) => handle_size(&mut input, &mut gas_used, ctx),
        Calls::addressExists(c) => handle_address_exists(&mut input, &mut gas_used, c.addr, ctx),
        Calls::lookup(c) => handle_lookup(&mut input, &mut gas_used, c.addr, ctx),
        Calls::lookupIndex(c) => handle_lookup_index(&mut input, &mut gas_used, c.index, ctx),
        Calls::register(c) => handle_register(&mut input, &mut gas_used, c.addr, ctx),
        Calls::compress(c) => handle_compress(&mut input, &mut gas_used, c.addr, ctx),
        Calls::decompress(c) => handle_decompress(&mut input, &mut gas_used, &c.buf, c.offset, ctx),
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

fn handle_size(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let size = arb_state
        .address_table
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;

    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(size).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_address_exists(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let exists = arb_state
        .address_table
        .address_exists(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let value = if exists { U256::from(1u64) } else { U256::ZERO };

    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_lookup(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let (index, exists) = arb_state
        .address_table
        .lookup(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    if !exists {
        crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_lookup_index(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    index_u256: U256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let index: u64 = index_u256
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let addr = match arb_state
        .address_table
        .lookup_index(internals, index)
        .map_err(ArbPrecompileError::fatal)?
    {
        Some(a) => a,
        None => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
    };

    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(addr.as_slice());

    crate::charge_storage_read(gas_used, ctx, 2 * SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_register(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let (index, already_registered) = arb_state
        .address_table
        .register(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;

    if already_registered {
        crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
        crate::charge_computation(gas_used, ctx, COPY_GAS);
        return Ok(PrecompileOutput::new(
            (*gas_used).min(gas_limit),
            U256::from(index).to_be_bytes::<32>().to_vec().into(),
        ));
    }

    // The index→address slot stores the address, so a zero address is a
    // zero-value write, charged the reset price.
    let index_write = if addr.is_zero() {
        SSTORE_ZERO_GAS
    } else {
        SSTORE_GAS
    };
    crate::charge_storage_read(gas_used, ctx, 2 * SLOAD_GAS);
    crate::charge_storage_write(gas_used, ctx, 2 * SSTORE_GAS + index_write);
    crate::charge_computation(gas_used, ctx, COPY_GAS);

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_compress(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
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
    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, result_words * COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        output.into(),
    ))
}

fn handle_decompress(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    buf: &Bytes,
    offset: U256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let ioffset: usize = offset
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    if ioffset >= buf.len() {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    let slice = &buf[ioffset..];

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
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

    // `body_sloads` counts the framework `OpenArbosState` SLOAD plus the body's
    // own reads; init already charged the framework SLOAD, so subtract one.
    let body_sloads: u64 = if raw_address { 0 } else { 2 };
    crate::charge_storage_read(gas_used, ctx, body_sloads * SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, 2 * COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        output.into(),
    ))
}
