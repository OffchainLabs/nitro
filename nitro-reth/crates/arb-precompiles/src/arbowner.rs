use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_primitives::multigas::NUM_RESOURCE_KIND;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::{
    address_set::AddressSet, arbos_state::arbos_from_input, burn::SystemBurner,
    programs::params::StylusParams,
};
use revm::{
    precompile::{PrecompileId, PrecompileOutput, PrecompileResult},
    primitives::Log,
};
use std::sync::Arc;

use crate::{interfaces::IArbOwner, ArbPrecompileError};

/// ArbOwner precompile address (0x70).
pub const ARBOWNER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x70,
]);

const ARBOS_VERSION_60: u64 = 60;

/// L1 pricer funds pool address.
const L1_PRICER_FUNDS_POOL_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xf6,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbowner_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbowner"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }
    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    verify_owner(&mut input, &mut gas_used)?;

    let call = match IArbOwner::ArbOwnerCalls::abi_decode(data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    gas_used = 0;
    crate::init_precompile_gas(&mut gas_used, data.len());

    use IArbOwner::ArbOwnerCalls as Calls;
    let is_read_only = matches!(
        call,
        Calls::getNetworkFeeAccount(_)
            | Calls::getInfraFeeAccount(_)
            | Calls::isChainOwner(_)
            | Calls::getAllChainOwners(_)
            | Calls::isTransactionFilterer(_)
            | Calls::getAllTransactionFilterers(_)
            | Calls::isNativeTokenOwner(_)
            | Calls::getAllNativeTokenOwners(_)
            | Calls::getFilteredFundsRecipient(_)
    );

    let result = match call {
        // Getters
        Calls::getNetworkFeeAccount(_) => handle_get_network_fee_account(&mut input, &mut gas_used),
        Calls::getInfraFeeAccount(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 5, 0) {
                return r;
            }
            handle_get_infra_fee_account(&mut input, &mut gas_used)
        }
        Calls::getFilteredFundsRecipient(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_get_filtered_funds_recipient(&mut input, &mut gas_used)
        }
        Calls::isChainOwner(_) => {
            handle_is_member(&mut input, &mut gas_used, AddressSetKind::ChainOwners)
        }
        Calls::getAllChainOwners(_) => {
            handle_get_all_members(&mut input, &mut gas_used, AddressSetKind::ChainOwners)
        }
        Calls::getAllTransactionFilterers(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_get_all_members(
                &mut input,
                &mut gas_used,
                AddressSetKind::TransactionFilterers,
            )
        }
        Calls::getAllNativeTokenOwners(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 41, 0) {
                return r;
            }
            handle_get_all_members(&mut input, &mut gas_used, AddressSetKind::NativeTokenOwners)
        }
        Calls::isTransactionFilterer(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_is_member(
                &mut input,
                &mut gas_used,
                AddressSetKind::TransactionFilterers,
            )
        }
        Calls::isNativeTokenOwner(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 41, 0) {
                return r;
            }
            handle_is_member(&mut input, &mut gas_used, AddressSetKind::NativeTokenOwners)
        }

        // Chain owner management
        Calls::addChainOwner(_) => handle_add_chain_owner(&mut input, &mut gas_used),
        Calls::removeChainOwner(_) => handle_remove_chain_owner(&mut input, &mut gas_used),

        // Root state setters
        Calls::setNetworkFeeAccount(_) => handle_set_network_fee_account(&mut input, &mut gas_used),
        Calls::setInfraFeeAccount(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 5, 0) {
                return r;
            }
            handle_set_infra_fee_account(&mut input, &mut gas_used)
        }
        Calls::setBrotliCompressionLevel(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            handle_set_brotli_compression_level(&mut input, &mut gas_used)
        }
        Calls::scheduleArbOSUpgrade(_) => handle_schedule_upgrade(&mut input, &mut gas_used),

        // L2 pricing setters
        Calls::setSpeedLimit(_) => match data.get(4..36) {
            None => Err(ArbPrecompileError::empty_revert(gas_used).into()),
            Some(bytes) if U256::from_be_slice(bytes).is_zero() => {
                Err(ArbPrecompileError::empty_revert(gas_used).into())
            }
            Some(_) => handle_set_speed_limit(&mut input, &mut gas_used),
        },
        Calls::setL2BaseFee(_) => handle_set_l2_base_fee(&mut input, &mut gas_used),
        Calls::setMinimumL2BaseFee(_) => handle_set_min_l2_base_fee(&mut input, &mut gas_used),
        Calls::setMaxBlockGasLimit(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            handle_set_max_block_gas_limit(&mut input, &mut gas_used)
        }
        Calls::setMaxTxGasLimit(_) => handle_set_max_tx_gas_limit(&mut input, &mut gas_used),
        Calls::setL2GasPricingInertia(_) => match data.get(4..36) {
            None => Err(ArbPrecompileError::empty_revert(gas_used).into()),
            Some(bytes) if U256::from_be_slice(bytes).is_zero() => {
                Err(ArbPrecompileError::empty_revert(gas_used).into())
            }
            Some(_) => handle_set_l2_pricing_inertia(&mut input, &mut gas_used),
        },
        Calls::setL2GasBacklogTolerance(_) => {
            handle_set_l2_backlog_tolerance(&mut input, &mut gas_used)
        }
        Calls::setGasBacklog(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            handle_set_gas_backlog(&mut input, &mut gas_used)
        }

        // L1 pricing setters
        Calls::setL1PricingEquilibrationUnits(_) => {
            handle_set_l1_equilibration_units(&mut input, &mut gas_used)
        }
        Calls::setL1PricingInertia(_) | Calls::setL1BaseFeeEstimateInertia(_) => {
            handle_set_l1_inertia(&mut input, &mut gas_used)
        }
        Calls::setL1PricingRewardRecipient(_) => {
            handle_set_l1_pay_rewards_to(&mut input, &mut gas_used)
        }
        Calls::setL1PricingRewardRate(_) => {
            handle_set_l1_per_unit_reward(&mut input, &mut gas_used)
        }
        Calls::setL1PricePerUnit(_) => handle_set_l1_price_per_unit(&mut input, &mut gas_used),
        Calls::setParentGasFloorPerToken(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            handle_set_parent_gas_floor_per_token(&mut input, &mut gas_used)
        }
        Calls::setPerBatchGasCharge(_) => handle_set_per_batch_gas_cost(&mut input, &mut gas_used),
        Calls::setAmortizedCostCapBips(_) => {
            handle_set_amortized_cost_cap_bips(&mut input, &mut gas_used)
        }
        Calls::releaseL1PricerSurplusFunds(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 10, 0) {
                return r;
            }
            handle_release_l1_pricer_surplus_funds(&mut input, &mut gas_used)
        }

        // Stylus/Wasm parameter setters (all require ArbOS >= 30)
        Calls::setInkPrice(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            match read_u32_param(gas_used, data) {
                Err(e) => Err(e.into()),
                Ok(val) if val == 0 || val > 0xFF_FFFF => {
                    Err(ArbPrecompileError::empty_revert(gas_used).into())
                }
                Ok(val) => write_stylus_param(&mut input, &mut gas_used, |p| p.ink_price = val),
            }
        }
        Calls::setWasmMaxStackDepth(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.max_stack_depth = val)
        }
        Calls::setWasmFreePages(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.free_pages = val as u16)
        }
        Calls::setWasmPageGas(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.page_gas = val as u16)
        }
        Calls::setWasmPageLimit(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.page_limit = val as u16)
        }
        Calls::setWasmMinInitGas(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.min_init_gas = val as u8)
        }
        Calls::setWasmInitCostScalar(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            // Stored as DivCeil(percent, 2); the reader multiplies by 2.
            let stored = (val as u64).saturating_add(1) / 2;
            write_stylus_param(&mut input, &mut gas_used, |p| {
                p.init_cost_scalar = stored as u8
            })
        }
        Calls::setWasmExpiryDays(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.expiry_days = val as u16)
        }
        Calls::setWasmKeepaliveDays(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.keepalive_days = val as u16)
        }
        Calls::setWasmBlockCacheSize(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| {
                p.block_cache_size = val as u16
            })
        }
        Calls::setWasmMaxSize(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 40, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| p.max_wasm_size = val)
        }
        Calls::setWasmActivationGas(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                gas_limit,
                arb_chainspec::arbos_version::ARBOS_VERSION_59,
                0,
            ) {
                return r;
            }
            if data.len() < 36 {
                return crate::burn_all_revert(gas_limit);
            }
            let val = U256::from_be_slice(&data[4..36]);
            handle_set_activation_gas(&mut input, &mut gas_used, val)
        }
        Calls::setMaxStylusContractFragments(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            let val = read_u32_param(gas_used, data)?;
            write_stylus_param(&mut input, &mut gas_used, |p| {
                p.max_fragment_count = val as u8
            })
        }
        Calls::addWasmCacheManager(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            handle_add_cache_manager(&mut input, &mut gas_used)
        }
        Calls::removeWasmCacheManager(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 30, 0) {
                return r;
            }
            handle_remove_cache_manager(&mut input, &mut gas_used)
        }
        Calls::setCalldataPriceIncrease(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 40, 0) {
                return r;
            }
            handle_set_calldata_price_increase(&mut input, &mut gas_used)
        }
        Calls::setCollectTips(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, ARBOS_VERSION_60, 0) {
                return r;
            }
            handle_set_collect_tips(&mut input, &mut gas_used)
        }

        // Transaction filtering (all ArbOS >= 60)
        Calls::addTransactionFilterer(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_add_to_set_with_feature_check(
                &mut input,
                &mut gas_used,
                AddressSetKind::TransactionFilterers,
                FeatureTimeKind::TransactionFiltering,
                Some(IArbOwner::TransactionFiltererAdded::SIGNATURE_HASH),
            )
        }
        Calls::removeTransactionFilterer(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_remove_from_set(
                &mut input,
                &mut gas_used,
                AddressSetKind::TransactionFilterers,
                Some(IArbOwner::TransactionFiltererRemoved::SIGNATURE_HASH),
            )
        }
        Calls::setTransactionFilteringFrom(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_set_feature_time(
                &mut input,
                &mut gas_used,
                FeatureTimeKind::TransactionFiltering,
            )
        }
        Calls::setFilteredFundsRecipient(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_set_filtered_funds_recipient(&mut input, &mut gas_used)
        }

        // Native token management (all ArbOS >= 41)
        Calls::setNativeTokenManagementFrom(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 41, 0) {
                return r;
            }
            handle_set_feature_time(&mut input, &mut gas_used, FeatureTimeKind::NativeToken)
        }
        Calls::addNativeTokenOwner(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 41, 0) {
                return r;
            }
            handle_add_to_set_with_feature_check(
                &mut input,
                &mut gas_used,
                AddressSetKind::NativeTokenOwners,
                FeatureTimeKind::NativeToken,
                Some(IArbOwner::NativeTokenOwnerAdded::SIGNATURE_HASH),
            )
        }
        Calls::removeNativeTokenOwner(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 41, 0) {
                return r;
            }
            handle_remove_from_set(
                &mut input,
                &mut gas_used,
                AddressSetKind::NativeTokenOwners,
                Some(IArbOwner::NativeTokenOwnerRemoved::SIGNATURE_HASH),
            )
        }

        // Gas pricing constraints
        Calls::setGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            handle_set_gas_pricing_constraints(&mut input, &mut gas_used)
        }
        Calls::setMultiGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_set_multi_gas_pricing_constraints(&mut input, &mut gas_used)
        }

        // Chain config (ArbOS >= 11)
        Calls::setChainConfig(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 11, 0) {
                return r;
            }
            handle_set_chain_config(&mut input, &mut gas_used)
        }
    };

    let result = match result {
        Ok(output) => {
            if output.reverted {
                Ok(PrecompileOutput::new_reverted(0, output.bytes))
            } else {
                let arbos_version = ctx.block.arbos_version;
                if !is_read_only || arbos_version < 11 {
                    emit_owner_acts(&mut input, &selector, data);
                }
                Ok(PrecompileOutput::new(0, output.bytes))
            }
        }
        Err(_) => Ok(PrecompileOutput::new_reverted(0, Default::default())),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

// Owner verification

fn verify_owner(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> Result<(), ArbPrecompileError> {
    let caller = input.caller;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let is_owner = arb_state
        .chain_owners
        .is_member(internals, caller)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS);
    if !is_owner {
        return Err(ArbPrecompileError::empty_revert(*gas_used));
    }
    Ok(())
}

// Storage helpers

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

#[derive(Clone, Copy)]
enum AddressSetKind {
    ChainOwners,
    NativeTokenOwners,
    TransactionFilterers,
}

#[derive(Clone, Copy)]
enum FeatureTimeKind {
    NativeToken,
    TransactionFiltering,
}

fn address_set<'a, D, B>(
    state: &'a arbos::arbos_state::ArbosState<'_, D, B>,
    kind: AddressSetKind,
) -> &'a AddressSet<'a, D>
where
    B: arbos::burn::Burner,
{
    match kind {
        AddressSetKind::ChainOwners => &state.chain_owners,
        AddressSetKind::NativeTokenOwners => &state.native_token_owners,
        AddressSetKind::TransactionFilterers => &state.transaction_filterers,
    }
}

fn read_feature_time<D, B, C>(
    state: &arbos::arbos_state::ArbosState<'_, D, B>,
    backend: &mut C,
    kind: FeatureTimeKind,
) -> Result<u64, ArbPrecompileError>
where
    B: arbos::burn::Burner,
    C: arb_storage::StorageBackend,
{
    match kind {
        FeatureTimeKind::NativeToken => state.native_token_management_from_time(backend),
        FeatureTimeKind::TransactionFiltering => state.transaction_filtering_from_time(backend),
    }
    .map_err(ArbPrecompileError::fatal)
}

fn write_feature_time<D, B, C>(
    state: &arbos::arbos_state::ArbosState<'_, D, B>,
    backend: &mut C,
    kind: FeatureTimeKind,
    value: u64,
) -> Result<(), ArbPrecompileError>
where
    B: arbos::burn::Burner,
    C: arb_storage::StorageBackend,
{
    match kind {
        FeatureTimeKind::NativeToken => state.set_native_token_management_from_time(backend, value),
        FeatureTimeKind::TransactionFiltering => {
            state.set_transaction_filtering_from_time(backend, value)
        }
    }
    .map_err(ArbPrecompileError::fatal)
}

fn field_read_output(gas_limit: u64, gas_used: u64, value: U256) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

// Root field readers

fn handle_get_network_fee_account(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .network_fee_account(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS);
    field_read_output(gas_limit, *gas_used, U256::from_be_slice(addr.as_slice()))
}

fn handle_get_infra_fee_account(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .infra_fee_account(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS);
    field_read_output(gas_limit, *gas_used, U256::from_be_slice(addr.as_slice()))
}

fn handle_get_filtered_funds_recipient(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let addr = arb_state
        .filtered_funds_recipient(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS);
    field_read_output(gas_limit, *gas_used, U256::from_be_slice(addr.as_slice()))
}

// Root field setters

fn handle_set_network_fee_account(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .set_network_fee_account(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_infra_fee_account(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .set_infra_fee_account(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_brotli_compression_level(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let level: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .set_brotli_compression_level(internals, level)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_filtered_funds_recipient(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .set_filtered_funds_recipient(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    emit_address_event(
        input,
        IArbOwner::FilteredFundsRecipientSet::SIGNATURE_HASH,
        addr,
    );
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_schedule_upgrade(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let new_version: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    let timestamp: u64 = U256::from_be_slice(&data[36..68]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .schedule_arbos_upgrade(internals, new_version, timestamp)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 2 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

// L2 pricing setters

fn handle_set_speed_limit(input: &mut PrecompileInput<'_>, gas_used: &mut u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_speed_limit_per_second(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l2_base_fee(input: &mut PrecompileInput<'_>, gas_used: &mut u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val = U256::from_be_slice(&data[4..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_base_fee_wei(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_min_l2_base_fee(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val = U256::from_be_slice(&data[4..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_min_base_fee_wei(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_max_block_gas_limit(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_max_per_block_gas_limit(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_max_tx_gas_limit(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_max_per_tx_gas_limit(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l2_pricing_inertia(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_pricing_inertia(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l2_backlog_tolerance(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_backlog_tolerance(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_gas_backlog(input: &mut PrecompileInput<'_>, gas_used: &mut u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .set_gas_backlog(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

// L1 pricing setters

fn handle_set_l1_equilibration_units(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val = U256::from_be_slice(&data[4..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_equilibration_units(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l1_inertia(input: &mut PrecompileInput<'_>, gas_used: &mut u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_inertia(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l1_pay_rewards_to(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_pay_rewards_to(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l1_per_unit_reward(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_per_unit_reward(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_l1_price_per_unit(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val = U256::from_be_slice(&data[4..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_price_per_unit(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_parent_gas_floor_per_token(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_parent_gas_floor_per_token(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_per_batch_gas_cost(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let raw = U256::from_be_slice(&data[4..36]);
    // setPerBatchGasCharge accepts an int64; reinterpret the low 64 bits as signed.
    let val_u64: u64 = raw.try_into().unwrap_or(u64::MAX);
    let val_i64 = val_u64 as i64;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_per_batch_gas_cost(internals, val_i64)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_amortized_cost_cap_bips(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let val: u64 = U256::from_be_slice(&data[4..36]).try_into().unwrap_or(0);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l1_pricing_state
        .set_amortized_cost_cap_bips(internals, val)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

// AddressSet handlers

fn handle_is_member(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    kind: AddressSetKind,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let is_member = address_set(&arb_state, kind)
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_all_members(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    kind: AddressSetKind,
) -> PrecompileResult {
    let gas_limit = input.gas;
    const MAX_MEMBERS: u64 = 65_536;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let members = address_set(&arb_state, kind)
        .all_members(internals, MAX_MEMBERS)
        .map_err(ArbPrecompileError::fatal)?;
    let count = members.len() as u64;

    let mut out = Vec::with_capacity(64 + 32 * members.len());
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for member in &members {
        let mut word = [0u8; 32];
        word[12..32].copy_from_slice(member.as_slice());
        out.extend_from_slice(&word);
    }

    let extra = (1 + count) * SLOAD_GAS + COPY_GAS;
    crate::charge_precompile_gas(gas_used, extra);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.into(),
    ))
}

fn handle_add_chain_owner(input: &mut PrecompileInput<'_>, gas_used: &mut u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();
    arb_state
        .chain_owners
        .add(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    if arbos_version >= 60 {
        emit_address_event(input, IArbOwner::ChainOwnerAdded::SIGNATURE_HASH, addr);
    }

    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_remove_chain_owner(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();
    if !arb_state
        .chain_owners
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?
    {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    arb_state
        .chain_owners
        .remove(internals, addr, arbos_version)
        .map_err(ArbPrecompileError::fatal)?;
    if arbos_version >= 60 {
        emit_address_event(input, IArbOwner::ChainOwnerRemoved::SIGNATURE_HASH, addr);
    }

    crate::charge_precompile_gas(gas_used, 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

/// surplus = pool_balance - recognized_fees; capped by maxWeiToRelease.
/// Adds the released amount to L1FeesAvailable rather than zeroing it.
fn handle_release_l1_pricer_surplus_funds(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let max_wei = U256::from_be_slice(&data[4..36]);

    let pool_balance = {
        let acct = input
            .internals_mut()
            .load_account(L1_PRICER_FUNDS_POOL_ADDRESS)
            .map_err(ArbPrecompileError::fatal)?;
        acct.data.info.balance
    };

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let recognized = arb_state
        .l1_pricing_state
        .l1_fees_available(internals)
        .map_err(ArbPrecompileError::fatal)?;

    if pool_balance <= recognized {
        crate::charge_precompile_gas(gas_used, SLOAD_GAS + COPY_GAS + 100);
        return Ok(PrecompileOutput::new(
            (*gas_used).min(gas_limit),
            U256::ZERO.to_be_bytes::<32>().to_vec().into(),
        ));
    }

    let mut wei_to_transfer = pool_balance - recognized;
    if wei_to_transfer > max_wei {
        wei_to_transfer = max_wei;
    }

    let new_available = recognized + wei_to_transfer;
    arb_state
        .l1_pricing_state
        .set_l1_fees_available(internals, new_available)
        .map_err(ArbPrecompileError::fatal)?;

    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SSTORE_GAS + COPY_GAS + 100);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        wei_to_transfer.to_be_bytes::<32>().to_vec().into(),
    ))
}

// Stylus parameter setters

fn write_stylus_param(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    mutate: impl FnOnce(&mut StylusParams),
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let mut params = arb_state
        .programs
        .params(internals)
        .map_err(ArbPrecompileError::fatal)?;
    mutate(&mut params);
    arb_state
        .programs
        .save_params(internals, &params)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_activation_gas(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    value: U256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let value_u64: u64 = value.try_into().unwrap_or(u64::MAX);
    arb_state
        .programs
        .set_activation_gas(internals, value_u64)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn read_u32_param(gas_used: u64, data: &[u8]) -> Result<u32, ArbPrecompileError> {
    if data.len() < 36 {
        return Err(ArbPrecompileError::empty_revert(gas_used));
    }
    let val = U256::from_be_slice(&data[4..36]);
    val.try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(gas_used))
}

// Cache manager helpers

fn handle_add_cache_manager(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .programs
        .cache_managers
        .add(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_remove_cache_manager(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();
    if !arb_state
        .programs
        .cache_managers
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?
    {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    arb_state
        .programs
        .cache_managers
        .remove(internals, addr, arbos_version)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

/// One week in seconds.
const FEATURE_ENABLE_DELAY: u64 = 7 * 24 * 60 * 60;

fn handle_set_feature_time(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    kind: FeatureTimeKind,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let timestamp: u64 = U256::from_be_slice(&data[4..36])
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    load_arbos(input)?;
    let now: u64 = input
        .internals_mut()
        .block_timestamp()
        .try_into()
        .unwrap_or(0u64);
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    if timestamp == 0 {
        write_feature_time(&arb_state, internals, kind, 0)?;
        crate::charge_precompile_gas(gas_used, SSTORE_GAS + COPY_GAS);
        return Ok(PrecompileOutput::new(
            (*gas_used).min(gas_limit),
            Vec::new().into(),
        ));
    }

    let stored = read_feature_time(&arb_state, internals, kind)?;

    if (stored > now + FEATURE_ENABLE_DELAY || stored == 0)
        && timestamp < now + FEATURE_ENABLE_DELAY
    {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    if stored > now && stored <= now + FEATURE_ENABLE_DELAY && timestamp < stored {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    write_feature_time(&arb_state, internals, kind, timestamp)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn emit_address_event(input: &mut PrecompileInput<'_>, topic0: B256, addr: Address) {
    let topic1 = B256::left_padding_from(addr.as_slice());
    input.internals_mut().log(Log::new_unchecked(
        ARBOWNER_ADDRESS,
        vec![topic0, topic1],
        alloy_primitives::Bytes::new(),
    ));
}

fn handle_add_to_set_with_feature_check(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    set_kind: AddressSetKind,
    feature_kind: FeatureTimeKind,
    event_topic: Option<B256>,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;
    let now: u64 = input
        .internals_mut()
        .block_timestamp()
        .try_into()
        .unwrap_or(0u64);
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    let enabled_time = read_feature_time(&arb_state, internals, feature_kind)?;
    if enabled_time == 0 || enabled_time > now {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    address_set(&arb_state, set_kind)
        .add(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;

    if let Some(topic0) = event_topic {
        emit_address_event(input, topic0, addr);
    }

    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_remove_from_set(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    set_kind: AddressSetKind,
    event_topic: Option<B256>,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();

    let set = address_set(&arb_state, set_kind);
    if !set
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?
    {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    set.remove(internals, addr, arbos_version)
        .map_err(ArbPrecompileError::fatal)?;

    if let Some(topic0) = event_topic {
        emit_address_event(input, topic0, addr);
    }

    crate::charge_precompile_gas(gas_used, 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

// Gas constraint helpers

const GAS_CONSTRAINTS_MAX_NUM: usize = 20;
const MAX_PRICING_EXPONENT_BIPS: u64 = 85_000;

fn handle_set_gas_pricing_constraints(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    // Minimum: selector(4) + offset(32) + length(32) = 68 bytes
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    let count: u64 = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    let expected_len = 68 + (count as usize) * 96;
    if data.len() < expected_len {
        return crate::burn_all_revert(gas_limit);
    }

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    arb_state
        .l2_pricing_state
        .clear_gas_constraints(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let arbos_version = arb_state.arbos_version();
    use arb_chainspec::arbos_version as arb_ver;
    if (arb_ver::ARBOS_VERSION_MULTI_CONSTRAINT_FIX..arb_ver::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS)
        .contains(&arbos_version)
        && (count as usize) > GAS_CONSTRAINTS_MAX_NUM
    {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    for i in 0..count {
        let base = 68 + (i as usize) * 96;
        let target: u64 = U256::from_be_slice(&data[base..base + 32])
            .try_into()
            .unwrap_or(0);
        let window: u64 = U256::from_be_slice(&data[base + 32..base + 64])
            .try_into()
            .unwrap_or(0);
        let backlog: u64 = U256::from_be_slice(&data[base + 64..base + 96])
            .try_into()
            .unwrap_or(0);

        if target == 0 || window == 0 {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }

        arb_state
            .l2_pricing_state
            .add_gas_constraint(internals, target, window, backlog)
            .map_err(ArbPrecompileError::fatal)?;
    }

    let extra = SLOAD_GAS + (count * 4 + 2) * SSTORE_GAS + count * SLOAD_GAS + COPY_GAS;
    crate::charge_precompile_gas(gas_used, extra);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

/// ABI: `setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])`.
/// Each struct has head layout: [resources offset, window_secs, target, backlog].
fn handle_set_multi_gas_pricing_constraints(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    let _outer_offset: usize = U256::from_be_slice(&data[4..36])
        .try_into()
        .unwrap_or(0usize);
    let count: u64 = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    let array_data_start = 68;

    let mut struct_offsets = Vec::with_capacity(count as usize);
    for i in 0..count as usize {
        let offset_pos = array_data_start + i * 32;
        if data.len() < offset_pos + 32 {
            return crate::burn_all_revert(gas_limit);
        }
        let offset: usize = U256::from_be_slice(&data[offset_pos..offset_pos + 32])
            .try_into()
            .unwrap_or(0);
        struct_offsets.push(array_data_start + offset);
    }

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    arb_state
        .l2_pricing_state
        .clear_multi_gas_constraints(internals)
        .map_err(ArbPrecompileError::fatal)?;

    for (i, &struct_start) in struct_offsets.iter().enumerate() {
        if data.len() < struct_start + 128 {
            return crate::burn_all_revert(gas_limit);
        }

        let resources_offset: usize = U256::from_be_slice(&data[struct_start..struct_start + 32])
            .try_into()
            .unwrap_or(0);
        let window: u32 = U256::from_be_slice(&data[struct_start + 32..struct_start + 64])
            .try_into()
            .unwrap_or(0);
        let target: u64 = U256::from_be_slice(&data[struct_start + 64..struct_start + 96])
            .try_into()
            .unwrap_or(0);
        let backlog: u64 = U256::from_be_slice(&data[struct_start + 96..struct_start + 128])
            .try_into()
            .unwrap_or(0);

        if target == 0 || window == 0 {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
        let resources_start = struct_start + resources_offset;

        if data.len() < resources_start + 32 {
            return crate::burn_all_revert(gas_limit);
        }

        let num_resources: usize =
            U256::from_be_slice(&data[resources_start..resources_start + 32])
                .try_into()
                .unwrap_or(0);

        let mut weights = [0u64; NUM_RESOURCE_KIND];
        for r in 0..num_resources {
            let r_start = resources_start + 32 + r * 64;
            if data.len() < r_start + 64 {
                return crate::burn_all_revert(gas_limit);
            }
            let resource: u8 = U256::from_be_slice(&data[r_start..r_start + 32])
                .try_into()
                .unwrap_or(0);
            let weight: u64 = U256::from_be_slice(&data[r_start + 32..r_start + 64])
                .try_into()
                .unwrap_or(0);

            if (resource as usize) < NUM_RESOURCE_KIND {
                weights[resource as usize] = weight;
            }
        }

        arb_state
            .l2_pricing_state
            .add_multi_gas_constraint(internals, target, window, backlog, &weights)
            .map_err(ArbPrecompileError::fatal)?;

        validate_multi_gas_exponents(internals, &arb_state, (i as u64) + 1, *gas_used)?;
    }

    let extra = (count * 16 + 2) * SSTORE_GAS + (count * 12 + 2) * SLOAD_GAS + COPY_GAS;
    crate::charge_precompile_gas(gas_used, extra);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn validate_multi_gas_exponents<D, B, C>(
    backend: &mut C,
    arb_state: &arbos::arbos_state::ArbosState<'_, D, B>,
    count: u64,
    gas_used: u64,
) -> Result<(), ArbPrecompileError>
where
    B: arbos::burn::Burner,
    C: arb_storage::StorageBackend,
{
    use arb_primitives::multigas::ResourceKind;
    let mut exponents = [0u64; NUM_RESOURCE_KIND];

    for i in 0..count {
        let constraint = arb_state.l2_pricing_state.open_multi_gas_constraint_at(i);
        let target = constraint
            .target(backend)
            .map_err(ArbPrecompileError::fatal)?;
        let backlog = constraint
            .backlog(backend)
            .map_err(ArbPrecompileError::fatal)?;

        if backlog == 0 {
            continue;
        }

        let window = constraint
            .adjustment_window(backend)
            .map_err(ArbPrecompileError::fatal)? as u64;
        let max_weight = constraint
            .max_weight(backend)
            .map_err(ArbPrecompileError::fatal)?;

        if max_weight == 0 || target == 0 || window == 0 {
            continue;
        }

        let divisor = (window as u128)
            .saturating_mul(target as u128)
            .saturating_mul(max_weight as u128);

        for (r, exponent) in exponents.iter_mut().enumerate().take(NUM_RESOURCE_KIND) {
            let kind = ResourceKind::ALL[r];
            let weight = constraint
                .resource_weight(backend, kind)
                .map_err(ArbPrecompileError::fatal)?;
            if weight == 0 {
                continue;
            }

            let dividend = (backlog as u128)
                .saturating_mul(weight as u128)
                .saturating_mul(10_000);
            let exp = if divisor > 0 {
                (dividend / divisor) as u64
            } else {
                0
            };
            *exponent = exponent.saturating_add(exp);
        }
    }

    for &exp in &exponents {
        if exp > MAX_PRICING_EXPONENT_BIPS {
            return Err(ArbPrecompileError::empty_revert(gas_used));
        }
    }

    Ok(())
}

// SetChainConfig

fn handle_set_chain_config(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    let bytes_len: usize = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| ArbPrecompileError::empty_revert(*gas_used))?;

    if data.len() < 68 + bytes_len {
        return crate::burn_all_revert(gas_limit);
    }
    let config_bytes = data[68..68 + bytes_len].to_vec();

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    let old_len = arb_state
        .chain_config(internals)
        .map_err(ArbPrecompileError::fatal)?
        .len() as u64;
    arb_state
        .set_chain_config(internals, &config_bytes)
        .map_err(ArbPrecompileError::fatal)?;

    let old_slots = old_len.div_ceil(32);
    let new_slots = (bytes_len as u64).div_ceil(32);
    let total_stores = old_slots + 1 + new_slots;
    let extra = total_stores * SSTORE_GAS + SLOAD_GAS + COPY_GAS;
    crate::charge_precompile_gas(gas_used, extra);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_calldata_price_increase(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let enabled = U256::from_be_slice(&data[4..36]) != U256::ZERO;

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .features
        .set_calldata_price_increase(internals, enabled)
        .map_err(ArbPrecompileError::fatal)?;

    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_set_collect_tips(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let enabled = U256::from_be_slice(&data[4..36]) != U256::ZERO;

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .set_collect_tips(internals, enabled)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SSTORE_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

/// Emit the OwnerActs event: OwnerActs(bytes4 method, address owner, bytes data).
fn emit_owner_acts(input: &mut PrecompileInput<'_>, selector: &[u8; 4], calldata: &[u8]) {
    let topic0 = IArbOwner::OwnerActs::SIGNATURE_HASH;
    let mut method_topic = [0u8; 32];
    method_topic[..4].copy_from_slice(selector);
    let topic1 = B256::from(method_topic);
    let topic2 = B256::left_padding_from(input.caller.as_slice());

    let mut log_data = Vec::with_capacity(64 + calldata.len().div_ceil(32) * 32);
    log_data.extend_from_slice(&U256::from(32).to_be_bytes::<32>());
    log_data.extend_from_slice(&U256::from(calldata.len()).to_be_bytes::<32>());
    log_data.extend_from_slice(calldata);
    let pad = (32 - (calldata.len() % 32)) % 32;
    log_data.extend(std::iter::repeat_n(0u8, pad));

    input.internals_mut().log(Log::new_unchecked(
        ARBOWNER_ADDRESS,
        vec![topic0, topic1, topic2],
        log_data.into(),
    ));
}
