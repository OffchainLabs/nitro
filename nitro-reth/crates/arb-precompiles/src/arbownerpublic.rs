use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::address_set::AddressSetError;
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbOwnerPublic, ArbPrecompileError};

/// ArbOwnerPublic precompile address (0x6b).
pub const ARBOWNERPUBLIC_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6b,
]);

const SLOAD_GAS: u64 = 800;
const WARM_SLOAD_GAS: u64 = 100;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbownerpublic_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbownerpublic"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbOwnerPublic::ArbOwnerPublicCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbOwnerPublic::ArbOwnerPublicCalls as Calls;
    let result = match call {
        Calls::getNetworkFeeAccount(_) => read_network_fee_account(&mut input, ctx),
        Calls::getInfraFeeAccount(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_5,
                0,
            ) {
                return r;
            }
            read_infra_fee_account(&mut input, ctx)
        }
        Calls::getBrotliCompressionLevel(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_20,
                0,
            ) {
                return r;
            }
            read_brotli_compression_level(&mut input, ctx)
        }
        Calls::getScheduledUpgrade(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_20,
                0,
            ) {
                return r;
            }
            handle_scheduled_upgrade(&mut input, ctx)
        }
        Calls::isChainOwner(c) => handle_is_chain_owner(&mut input, c.addr, ctx),
        Calls::getAllChainOwners(_) => handle_get_all_chain_owners(&mut input, ctx),
        Calls::rectifyChainOwner(c) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_11,
                0,
            ) {
                return r;
            }
            handle_rectify_chain_owner(&mut input, &mut gas_used, c.ownerToRectify, ctx)
        }
        Calls::isNativeTokenOwner(c) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_41,
                0,
            ) {
                return r;
            }
            handle_is_native_token_owner(&mut input, c.addr, ctx)
        }
        Calls::isTransactionFilterer(c) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
                0,
            ) {
                return r;
            }
            handle_is_transaction_filterer(&mut input, c.filterer, ctx)
        }
        Calls::getAllNativeTokenOwners(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_41,
                0,
            ) {
                return r;
            }
            handle_get_all_native_token_owners(&mut input, ctx)
        }
        Calls::getAllTransactionFilterers(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
                0,
            ) {
                return r;
            }
            handle_get_all_transaction_filterers(&mut input, ctx)
        }
        Calls::getNativeTokenManagementFrom(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_50,
                0,
            ) {
                return r;
            }
            read_native_token_management_from(&mut input, ctx)
        }
        Calls::getTransactionFilteringFrom(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
                0,
            ) {
                return r;
            }
            read_transaction_filtering_from(&mut input, ctx)
        }
        Calls::getFilteredFundsRecipient(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
                0,
            ) {
                return r;
            }
            read_filtered_funds_recipient(&mut input, ctx)
        }
        Calls::isCalldataPriceIncreaseEnabled(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_40,
                0,
            ) {
                return r;
            }
            handle_is_calldata_price_increase_enabled(&mut input, ctx)
        }
        Calls::getParentGasFloorPerToken(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_50,
                0,
            ) {
                return r;
            }
            read_parent_gas_floor_per_token(&mut input, ctx)
        }
        Calls::getMaxStylusContractFragments(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_60,
                0,
            ) {
                return r;
            }
            handle_max_stylus_fragments(&mut input, ctx)
        }
        Calls::getCollectTips(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_60,
                0,
            ) {
                return r;
            }
            handle_get_collect_tips(&mut input, &mut gas_used, ctx)
        }
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

fn field_read_output(gas_limit: u64, value: U256) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn read_network_fee_account(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .network_fee_account(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from_be_slice(addr.as_slice()))
}

fn read_infra_fee_account(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let addr = if ctx.block.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_6 {
        arb_state
            .network_fee_account(internals)
            .map_err(ArbPrecompileError::fatal)?
    } else {
        arb_state
            .infra_fee_account(internals)
            .map_err(ArbPrecompileError::fatal)?
    };
    field_read_output(gas_limit, U256::from_be_slice(addr.as_slice()))
}

fn read_brotli_compression_level(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = arb_state
        .brotli_compression_level(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_native_token_management_from(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = arb_state
        .native_token_management_from_time(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_transaction_filtering_from(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = arb_state
        .transaction_filtering_from_time(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_filtered_funds_recipient(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .filtered_funds_recipient(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from_be_slice(addr.as_slice()))
}

fn read_parent_gas_floor_per_token(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = arb_state
        .l1_pricing_state
        .parent_gas_floor_per_token(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn handle_scheduled_upgrade(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let (version, timestamp) = arb_state
        .get_scheduled_upgrade(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&U256::from(version).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(timestamp).to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_is_chain_owner(
    input: &mut PrecompileInput<'_>,
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
    let is_owner = arb_state
        .chain_owners
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let result = if is_owner {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_native_token_owner(
    input: &mut PrecompileInput<'_>,
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
    let is_member = arb_state
        .native_token_owners
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_transaction_filterer(
    input: &mut PrecompileInput<'_>,
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
    let is_member = arb_state
        .transaction_filterers
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

enum AddressSetKind {
    ChainOwners,
    NativeTokenOwners,
    TransactionFilterers,
}

fn handle_get_all_chain_owners(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    handle_get_all_set_members(input, AddressSetKind::ChainOwners, 256, ctx)
}

fn handle_get_all_native_token_owners(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    handle_get_all_set_members(input, AddressSetKind::NativeTokenOwners, 65_536, ctx)
}

fn handle_get_all_transaction_filterers(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    handle_get_all_set_members(input, AddressSetKind::TransactionFilterers, 65_536, ctx)
}

fn handle_get_all_set_members(
    input: &mut PrecompileInput<'_>,
    kind: AddressSetKind,
    cap: u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let set = match kind {
        AddressSetKind::ChainOwners => &arb_state.chain_owners,
        AddressSetKind::NativeTokenOwners => &arb_state.native_token_owners,
        AddressSetKind::TransactionFilterers => &arb_state.transaction_filterers,
    };
    let members = set
        .all_members(internals, cap)
        .map_err(ArbPrecompileError::fatal)?;
    let count = members.len() as u64;

    let mut out = Vec::with_capacity(64 + members.len() * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for member in &members {
        let mut word = [0u8; 32];
        word[12..32].copy_from_slice(member.as_slice());
        out.extend_from_slice(&word);
    }

    Ok(PrecompileOutput::new(
        ((2 + count) * SLOAD_GAS + (2 + count) * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_rectify_chain_owner(
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

    match arb_state.chain_owners.rectify_mapping(internals, addr) {
        Ok(()) => {}
        Err(AddressSetError::Storage(s)) => return Err(ArbPrecompileError::fatal(s).into()),
        Err(AddressSetError::NotMember | AddressSetError::MappingAlreadyConsistent) => {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into())
        }
    }

    let topic0 = alloy_primitives::keccak256("ChainOwnerRectified(address)");
    let addr_hash = alloy_primitives::B256::left_padding_from(addr.as_slice());
    input
        .internals_mut()
        .log(alloy_primitives::Log::new_unchecked(
            ARBOWNERPUBLIC_ADDRESS,
            vec![topic0],
            addr_hash.0.to_vec().into(),
        ));

    const SSTORE_ZERO_GAS: u64 = 5_000;
    const RECTIFY_EVENT_GAS: u64 = 1_006;
    let gas_cost =
        SLOAD_GAS + 7 * SLOAD_GAS + SSTORE_ZERO_GAS + 3 * SSTORE_GAS + RECTIFY_EVENT_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_cost.min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_is_calldata_price_increase_enabled(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let enabled = arb_state
        .features
        .is_increased_calldata_price_enabled(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = if enabled {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_max_stylus_fragments(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let params = arb_state
        .programs
        .params(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let mut count = params.max_fragment_count;
    if count == 0 {
        count = arbos::programs::params::INITIAL_MAX_FRAGMENT_COUNT;
    }
    let mut out = [0u8; 32];
    out[31] = count;
    Ok(PrecompileOutput::new(
        METHOD_GAS.min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_get_collect_tips(
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
    let value = arb_state
        .collect_tips(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let mut out = [0u8; 32];
    if value {
        out[31] = 1;
    }
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.to_vec().into(),
    ))
}
