use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, B256, U256};
use revm::{
    precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult},
    primitives::Log,
};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
    CACHE_MANAGERS_KEY, CHAIN_CONFIG_SUBSPACE, CHAIN_OWNER_SUBSPACE, FEATURES_SUBSPACE,
    FILTERED_FUNDS_RECIPIENT_OFFSET, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET, NATIVE_TOKEN_SUBSPACE, PROGRAMS_SUBSPACE,
    ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE, TX_FILTERING_ENABLED_FROM_TIME_OFFSET,
};

/// ArbOwner precompile address (0x70).
pub const ARBOWNER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x70,
]);

// ── Selectors ────────────────────────────────────────────────────────

// Getters (also on ArbOwner in Go, though most are on ArbOwnerPublic)
const GET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0x2d, 0x91, 0x25, 0xe9];
const GET_INFRA_FEE_ACCOUNT: [u8; 4] = [0xee, 0x95, 0xa8, 0x24];
const IS_CHAIN_OWNER: [u8; 4] = [0x26, 0xef, 0x7f, 0x68];
const GET_ALL_CHAIN_OWNERS: [u8; 4] = [0x51, 0x6b, 0x4e, 0x0f];
const ADD_CHAIN_OWNER: [u8; 4] = [0x48, 0x1f, 0x8d, 0xbf];
const REMOVE_CHAIN_OWNER: [u8; 4] = [0x87, 0x92, 0x70, 0x1a];
const SET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0xfc, 0xdd, 0xe2, 0xb4];
const SET_INFRA_FEE_ACCOUNT: [u8; 4] = [0x57, 0xf5, 0x85, 0xdb];
const SCHEDULE_ARBOS_UPGRADE: [u8; 4] = [0xe3, 0x88, 0xb3, 0x81];
const SET_BROTLI_COMPRESSION_LEVEL: [u8; 4] = [0x53, 0x99, 0x12, 0x6f];
const SET_CHAIN_CONFIG: [u8; 4] = [0xed, 0xa7, 0x32, 0x12];
const SET_SPEED_LIMIT: [u8; 4] = [0x4d, 0x7a, 0x06, 0x0d];
const SET_L2_BASE_FEE: [u8; 4] = [0xd9, 0x9b, 0xc8, 0x0e];
const SET_MINIMUM_L2_BASE_FEE: [u8; 4] = [0xa0, 0x18, 0x8c, 0xdb];
const SET_MAX_BLOCK_GAS_LIMIT: [u8; 4] = [0xae, 0x10, 0x5c, 0x80];
const SET_MAX_TX_GAS_LIMIT: [u8; 4] = [0x39, 0x67, 0x36, 0x11];
const SET_L2_GAS_PRICING_INERTIA: [u8; 4] = [0x3f, 0xd6, 0x2a, 0x29];
const SET_L2_GAS_BACKLOG_TOLERANCE: [u8; 4] = [0x19, 0x8e, 0x71, 0x57];
const SET_GAS_BACKLOG: [u8; 4] = [0x68, 0xfc, 0x80, 0x8a];
const SET_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0xcc, 0x0d, 0x55, 0x6a]; // setGasPricingConstraints(uint64[3][])
const SET_MULTI_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0x2b, 0x05, 0xbb, 0x39]; // setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])
const SET_L1_PRICING_EQUILIBRATION_UNITS: [u8; 4] = [0x15, 0x2d, 0xb6, 0x96];
const SET_L1_PRICING_INERTIA: [u8; 4] = [0x77, 0x5a, 0x82, 0xe9];
const SET_L1_PRICING_REWARD_RECIPIENT: [u8; 4] = [0x93, 0x4b, 0xe0, 0x7d];
const SET_L1_PRICING_REWARD_RATE: [u8; 4] = [0xf6, 0x73, 0x95, 0x00];
const SET_L1_PRICE_PER_UNIT: [u8; 4] = [0x2b, 0x35, 0x2f, 0xae];
const SET_PARENT_GAS_FLOOR_PER_TOKEN: [u8; 4] = [0x3a, 0x93, 0x0b, 0x0b];
const SET_PER_BATCH_GAS_CHARGE: [u8; 4] = [0xfa, 0xd7, 0xf2, 0x0b];
const SET_AMORTIZED_COST_CAP_BIPS: [u8; 4] = [0x56, 0x19, 0x1c, 0xc3];
const RELEASE_L1_PRICER_SURPLUS_FUNDS: [u8; 4] = [0x31, 0x4b, 0xcf, 0x05];
const SET_L1_BASEFEE_ESTIMATE_INERTIA: [u8; 4] = [0x71, 0x8f, 0x78, 0x05];
const SET_INK_PRICE: [u8; 4] = [0x8c, 0x1d, 0x4f, 0xda];
const SET_WASM_MAX_STACK_DEPTH: [u8; 4] = [0x45, 0x67, 0xcc, 0x8e];
const SET_WASM_FREE_PAGES: [u8; 4] = [0x3f, 0x37, 0xa8, 0x46];
const SET_WASM_PAGE_GAS: [u8; 4] = [0xaa, 0xa6, 0x19, 0xe0];
const SET_WASM_PAGE_LIMIT: [u8; 4] = [0x65, 0x95, 0x38, 0x1a];
const SET_WASM_MIN_INIT_GAS: [u8; 4] = [0x82, 0x93, 0x40, 0x5e]; // setWasmMinInitGas(uint8,uint16)
const SET_WASM_INIT_COST_SCALAR: [u8; 4] = [0x67, 0xe0, 0x71, 0x8f];
const SET_WASM_EXPIRY_DAYS: [u8; 4] = [0xaa, 0xc6, 0x80, 0x18];
const SET_WASM_KEEPALIVE_DAYS: [u8; 4] = [0x2a, 0x9c, 0xbe, 0x3e];
const SET_WASM_BLOCK_CACHE_SIZE: [u8; 4] = [0x38, 0x0f, 0x14, 0x57];
const SET_WASM_MAX_SIZE: [u8; 4] = [0x45, 0x5e, 0xc2, 0xeb];
const ADD_WASM_CACHE_MANAGER: [u8; 4] = [0xff, 0xdc, 0xa5, 0x15];
const REMOVE_WASM_CACHE_MANAGER: [u8; 4] = [0xbf, 0x19, 0x73, 0x22];
const SET_MAX_STYLUS_CONTRACT_FRAGMENTS: [u8; 4] = [0xf1, 0xfe, 0x1a, 0x70];
const SET_CALLDATA_PRICE_INCREASE: [u8; 4] = [0x8e, 0xb9, 0x11, 0xd9]; // setCalldataPriceIncrease(bool)
const ADD_TRANSACTION_FILTERER: [u8; 4] = [0x59, 0xc8, 0x7a, 0xcc]; // addTransactionFilterer(address)
const REMOVE_TRANSACTION_FILTERER: [u8; 4] = [0x67, 0xad, 0xa0, 0x89]; // removeTransactionFilterer(address)
const GET_ALL_TRANSACTION_FILTERERS: [u8; 4] = [0x59, 0x5f, 0xbb, 0x5a]; // getAllTransactionFilterers()
const IS_TRANSACTION_FILTERER: [u8; 4] = [0xb3, 0x23, 0x52, 0xc3]; // isTransactionFilterer(address)
const SET_TRANSACTION_FILTERING_FROM: [u8; 4] = [0x46, 0x06, 0x6e, 0x45]; // setTransactionFilteringFrom(uint64)
const SET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0xb7, 0x9d, 0xa0, 0xe9]; // setFilteredFundsRecipient(address)
const GET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0x3c, 0xaa, 0x5f, 0x12]; // getFilteredFundsRecipient()
const SET_NATIVE_TOKEN_MANAGEMENT_FROM: [u8; 4] = [0xbd, 0xb8, 0xf7, 0x07]; // setNativeTokenManagementFrom(uint64)
const ADD_NATIVE_TOKEN_OWNER: [u8; 4] = [0xae, 0xb3, 0xa4, 0x64]; // addNativeTokenOwner(address)
const REMOVE_NATIVE_TOKEN_OWNER: [u8; 4] = [0x96, 0xa3, 0x75, 0x1d]; // removeNativeTokenOwner(address)
const GET_ALL_NATIVE_TOKEN_OWNERS: [u8; 4] = [0x3f, 0x86, 0x01, 0xe4]; // getAllNativeTokenOwners()
const IS_NATIVE_TOKEN_OWNER: [u8; 4] = [0xc6, 0x86, 0xf4, 0xdb]; // isNativeTokenOwner(address)

// ArbOS state offsets
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;

// L1 pricing field offsets
const L1_PAY_REWARDS_TO: u64 = 0;
const L1_EQUILIBRATION_UNITS: u64 = 1;
const L1_INERTIA: u64 = 2;
const L1_PER_UNIT_REWARD: u64 = 3;
const L1_PRICE_PER_UNIT: u64 = 7;
const L1_PER_BATCH_GAS_COST: u64 = 9;
const L1_AMORTIZED_COST_CAP_BIPS: u64 = 10;
const L1_FEES_AVAILABLE: u64 = 11;
const L1_GAS_FLOOR_PER_TOKEN: u64 = 12;

// L2 pricing field offsets
const L2_SPEED_LIMIT: u64 = 0;
const L2_PER_BLOCK_GAS_LIMIT: u64 = 1;
const L2_BASE_FEE: u64 = 2;
const L2_MIN_BASE_FEE: u64 = 3;
const L2_GAS_BACKLOG: u64 = 4;
const L2_PRICING_INERTIA: u64 = 5;
const L2_BACKLOG_TOLERANCE: u64 = 6;
const L2_PER_TX_GAS_LIMIT: u64 = 7;

/// L1 pricer funds pool address.
const L1_PRICER_FUNDS_POOL_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xf6,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbowner_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbowner"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }

    // Verify the caller is a chain owner.
    verify_owner(&mut input)?;

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        // ── Getters ──────────────────────────────────────────────
        GET_NETWORK_FEE_ACCOUNT => read_root_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        // GetInfraFeeAccount: ArbOS >= 5
        GET_INFRA_FEE_ACCOUNT => {
            if let Some(r) = crate::check_method_version(5, 0) {
                return r;
            }
            read_root_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET)
        }
        // GetFilteredFundsRecipient: ArbOS >= 60
        GET_FILTERED_FUNDS_RECIPIENT => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            read_root_field(&mut input, FILTERED_FUNDS_RECIPIENT_OFFSET)
        }
        IS_CHAIN_OWNER => handle_is_chain_owner(&mut input),
        GET_ALL_CHAIN_OWNERS => handle_get_all_chain_owners(&mut input),
        // GetAllTransactionFilterers: ArbOS >= 60
        GET_ALL_TRANSACTION_FILTERERS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_get_all_members(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        // GetAllNativeTokenOwners: ArbOS >= 41
        GET_ALL_NATIVE_TOKEN_OWNERS => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_get_all_members(&mut input, NATIVE_TOKEN_SUBSPACE)
        }
        // IsTransactionFilterer: ArbOS >= 60
        IS_TRANSACTION_FILTERER => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_is_member(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        // IsNativeTokenOwner: ArbOS >= 41
        IS_NATIVE_TOKEN_OWNER => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_is_member(&mut input, NATIVE_TOKEN_SUBSPACE)
        }

        // ── Chain owner management ─────────────────────────────────
        ADD_CHAIN_OWNER => handle_add_chain_owner(&mut input),
        REMOVE_CHAIN_OWNER => handle_remove_chain_owner(&mut input),

        // ── Root state setters ───────────────────────────────────
        SET_NETWORK_FEE_ACCOUNT => write_root_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        // SetInfraFeeAccount: ArbOS >= 5
        SET_INFRA_FEE_ACCOUNT => {
            if let Some(r) = crate::check_method_version(5, 0) {
                return r;
            }
            write_root_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET)
        }
        // SetBrotliCompressionLevel: ArbOS >= 20
        SET_BROTLI_COMPRESSION_LEVEL => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            write_root_field(&mut input, BROTLI_COMPRESSION_LEVEL_OFFSET)
        }
        SCHEDULE_ARBOS_UPGRADE => handle_schedule_upgrade(&mut input),

        // ── L2 pricing setters ───────────────────────────────────
        SET_SPEED_LIMIT => {
            let val = U256::from_be_slice(
                input
                    .data
                    .get(4..36)
                    .ok_or_else(|| PrecompileError::other("input too short"))?,
            );
            if val.is_zero() {
                return Err(PrecompileError::other("speed limit must be nonzero"));
            }
            write_l2_field(&mut input, L2_SPEED_LIMIT)
        }
        SET_L2_BASE_FEE => write_l2_field(&mut input, L2_BASE_FEE),
        SET_MINIMUM_L2_BASE_FEE => write_l2_field(&mut input, L2_MIN_BASE_FEE),
        // SetMaxBlockGasLimit: ArbOS >= 50
        SET_MAX_BLOCK_GAS_LIMIT => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            write_l2_field(&mut input, L2_PER_BLOCK_GAS_LIMIT)
        }
        SET_MAX_TX_GAS_LIMIT => {
            // ArbOS < 50: write to per-block gas limit; >= 50: per-tx gas limit.
            let version_slot = root_slot(0); // VERSION_OFFSET
            load_arbos(&mut input)?;
            let raw_version = sload_field(&mut input, version_slot)?.to::<u64>();
            let offset = if raw_version < 50 {
                L2_PER_BLOCK_GAS_LIMIT
            } else {
                L2_PER_TX_GAS_LIMIT
            };
            write_l2_field(&mut input, offset)
        }
        SET_L2_GAS_PRICING_INERTIA => {
            let val = U256::from_be_slice(
                input
                    .data
                    .get(4..36)
                    .ok_or_else(|| PrecompileError::other("input too short"))?,
            );
            if val.is_zero() {
                return Err(PrecompileError::other("price inertia must be nonzero"));
            }
            write_l2_field(&mut input, L2_PRICING_INERTIA)
        }
        SET_L2_GAS_BACKLOG_TOLERANCE => write_l2_field(&mut input, L2_BACKLOG_TOLERANCE),
        // SetGasBacklog: ArbOS >= 50
        SET_GAS_BACKLOG => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            write_l2_field(&mut input, L2_GAS_BACKLOG)
        }

        // ── L1 pricing setters ───────────────────────────────────
        SET_L1_PRICING_EQUILIBRATION_UNITS => write_l1_field(&mut input, L1_EQUILIBRATION_UNITS),
        SET_L1_PRICING_INERTIA => write_l1_field(&mut input, L1_INERTIA),
        SET_L1_PRICING_REWARD_RECIPIENT => write_l1_field(&mut input, L1_PAY_REWARDS_TO),
        SET_L1_PRICING_REWARD_RATE => write_l1_field(&mut input, L1_PER_UNIT_REWARD),
        SET_L1_PRICE_PER_UNIT => write_l1_field(&mut input, L1_PRICE_PER_UNIT),
        // SetParentGasFloorPerToken: ArbOS >= 50
        SET_PARENT_GAS_FLOOR_PER_TOKEN => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            write_l1_field(&mut input, L1_GAS_FLOOR_PER_TOKEN)
        }
        SET_PER_BATCH_GAS_CHARGE => write_l1_field(&mut input, L1_PER_BATCH_GAS_COST),
        SET_AMORTIZED_COST_CAP_BIPS => write_l1_field(&mut input, L1_AMORTIZED_COST_CAP_BIPS),
        SET_L1_BASEFEE_ESTIMATE_INERTIA => write_l1_field(&mut input, L1_INERTIA),
        // ReleaseL1PricerSurplusFunds: ArbOS >= 10
        RELEASE_L1_PRICER_SURPLUS_FUNDS => {
            if let Some(r) = crate::check_method_version(10, 0) {
                return r;
            }
            handle_release_l1_pricer_surplus_funds(&mut input)
        }

        // ── Stylus/Wasm parameter setters (all require ArbOS >= 30) ──
        SET_INK_PRICE => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            if val == 0 || val > 0xFF_FFFF {
                return Err(PrecompileError::other(
                    "ink price must be a positive uint24",
                ));
            }
            write_stylus_param(&mut input, StylusField::InkPrice, val as u64)
        }
        SET_WASM_MAX_STACK_DEPTH => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::MaxStackDepth, val as u64)
        }
        SET_WASM_FREE_PAGES => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::FreePages, val as u64)
        }
        SET_WASM_PAGE_GAS => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::PageGas, val as u64)
        }
        SET_WASM_PAGE_LIMIT => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::PageLimit, val as u64)
        }
        SET_WASM_MIN_INIT_GAS => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::MinInitGas, val as u64)
        }
        SET_WASM_INIT_COST_SCALAR => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::InitCostScalar, val as u64)
        }
        SET_WASM_EXPIRY_DAYS => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::ExpiryDays, val as u64)
        }
        SET_WASM_KEEPALIVE_DAYS => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::KeepaliveDays, val as u64)
        }
        SET_WASM_BLOCK_CACHE_SIZE => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::BlockCacheSize, val as u64)
        }
        // SetWasmMaxSize: ArbOS >= 40
        SET_WASM_MAX_SIZE => {
            if let Some(r) = crate::check_method_version(40, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::MaxWasmSize, val as u64)
        }
        // SetMaxStylusContractFragments: ArbOS >= 60
        SET_MAX_STYLUS_CONTRACT_FRAGMENTS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            let val = read_u32_param(data)?;
            write_stylus_param(&mut input, StylusField::MaxFragmentCount, val as u64)
        }
        // AddWasmCacheManager: ArbOS >= 30
        ADD_WASM_CACHE_MANAGER => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            handle_add_cache_manager(&mut input)
        }
        // RemoveWasmCacheManager: ArbOS >= 30
        REMOVE_WASM_CACHE_MANAGER => {
            if let Some(r) = crate::check_method_version(30, 0) {
                return r;
            }
            handle_remove_cache_manager(&mut input)
        }
        // SetCalldataPriceIncrease: ArbOS >= 40
        SET_CALLDATA_PRICE_INCREASE => {
            if let Some(r) = crate::check_method_version(40, 0) {
                return r;
            }
            handle_set_calldata_price_increase(&mut input)
        }

        // ── Transaction filtering (all ArbOS >= 60) ──────────────
        ADD_TRANSACTION_FILTERER => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_add_to_set_with_feature_check(
                &mut input,
                TRANSACTION_FILTERER_SUBSPACE,
                TX_FILTERING_ENABLED_FROM_TIME_OFFSET,
            )
        }
        REMOVE_TRANSACTION_FILTERER => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_remove_from_set(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        SET_TRANSACTION_FILTERING_FROM => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_set_feature_time(&mut input, TX_FILTERING_ENABLED_FROM_TIME_OFFSET)
        }
        // SetFilteredFundsRecipient: ArbOS >= 60
        SET_FILTERED_FUNDS_RECIPIENT => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            write_root_field(&mut input, FILTERED_FUNDS_RECIPIENT_OFFSET)
        }

        // ── Native token management (all ArbOS >= 41) ─────────────
        SET_NATIVE_TOKEN_MANAGEMENT_FROM => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_set_feature_time(&mut input, NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET)
        }
        ADD_NATIVE_TOKEN_OWNER => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_add_to_set_with_feature_check(
                &mut input,
                NATIVE_TOKEN_SUBSPACE,
                NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET,
            )
        }
        REMOVE_NATIVE_TOKEN_OWNER => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_remove_from_set(&mut input, NATIVE_TOKEN_SUBSPACE)
        }

        // ── Gas pricing constraints ──────────────────────────────
        // SetGasPricingConstraints: ArbOS >= 50
        SET_GAS_PRICING_CONSTRAINTS => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            handle_set_gas_pricing_constraints(&mut input)
        }
        // SetMultiGasPricingConstraints: ArbOS >= 60
        SET_MULTI_GAS_PRICING_CONSTRAINTS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_set_multi_gas_pricing_constraints(&mut input)
        }

        // ── Chain config (ArbOS >= 11) ──────────────────────────
        SET_CHAIN_CONFIG => {
            if let Some(r) = crate::check_method_version(11, 0) {
                return r;
            }
            handle_set_chain_config(&mut input)
        }

        _ => return crate::burn_all_revert(gas_limit),
    };
    // OwnerPrecompile wrapper: all successful calls are free (gas_used = 0).
    // Emit OwnerActs event on success. In Nitro, this is automatic for all
    // owner-only calls. For ArbOS < 11: emit for ALL calls (read+write).
    // For ArbOS >= 11: emit only for write calls (not read-only getters).
    let result = result.map(|output| {
        let arbos_version = crate::get_arbos_version();
        let is_read_only = matches!(
            selector,
            GET_NETWORK_FEE_ACCOUNT
                | GET_INFRA_FEE_ACCOUNT
                | IS_CHAIN_OWNER
                | GET_ALL_CHAIN_OWNERS
                | IS_TRANSACTION_FILTERER
                | GET_ALL_TRANSACTION_FILTERERS
                | IS_NATIVE_TOKEN_OWNER
                | GET_ALL_NATIVE_TOKEN_OWNERS
                | GET_FILTERED_FUNDS_RECIPIENT
        );
        if !is_read_only || arbos_version < 11 {
            emit_owner_acts(&mut input, &selector, data);
        }
        PrecompileOutput::new(0, output.bytes)
    });
    crate::gas_check(gas_limit, result)
}

// ── Owner verification ───────────────────────────────────────────────

fn verify_owner(input: &mut PrecompileInput<'_>) -> Result<(), PrecompileError> {
    let caller = input.caller;
    load_arbos(input)?;

    // Chain owners are stored in an AddressSet in the CHAIN_OWNER_SUBSPACE.
    // AddressSet.byAddress is at sub-storage key [0] within the set's storage.
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);

    let addr_as_b256 = alloy_primitives::B256::left_padding_from(caller.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_as_b256);

    let value = sload_field(input, member_slot)?;
    crate::charge_precompile_gas(800); // IsMember sload
    if value == U256::ZERO {
        return Err(PrecompileError::other(
            "ArbOwner: caller is not a chain owner",
        ));
    }
    Ok(())
}

// ── Storage helpers ──────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    Ok(())
}

fn sload_field(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let val = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    crate::charge_precompile_gas(SLOAD_GAS);
    Ok(val.data)
}

fn sstore_field(
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

fn read_root_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    let value = sload_field(input, root_slot(offset))?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn write_root_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let value = U256::from_be_slice(&data[4..36]);
    sstore_field(input, root_slot(offset), value)?;
    Ok(PrecompileOutput::new(
        (SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

fn write_l1_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let value = U256::from_be_slice(&data[4..36]);
    let field_slot = subspace_slot(L1_PRICING_SUBSPACE, offset);
    sstore_field(input, field_slot, value)?;
    Ok(PrecompileOutput::new(
        (SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

fn write_l2_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let value = U256::from_be_slice(&data[4..36]);
    let field_slot = subspace_slot(L2_PRICING_SUBSPACE, offset);
    sstore_field(input, field_slot, value)?;
    Ok(PrecompileOutput::new(
        (SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_schedule_upgrade(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let new_version = U256::from_be_slice(&data[4..36]);
    let timestamp = U256::from_be_slice(&data[36..68]);
    sstore_field(input, root_slot(UPGRADE_VERSION_OFFSET), new_version)?;
    sstore_field(input, root_slot(UPGRADE_TIMESTAMP_OFFSET), timestamp)?;
    Ok(PrecompileOutput::new(
        (2 * SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

/// Emit the OwnerActs event: OwnerActs(bytes4 method, address owner, bytes data).
/// Matches Nitro's automatic OwnerActs emission for all owner-only calls.
fn emit_owner_acts(input: &mut PrecompileInput<'_>, selector: &[u8; 4], calldata: &[u8]) {
    use alloy_primitives::{keccak256, Log, B256};

    // event OwnerActs(bytes4 indexed method, address indexed owner, bytes data)
    let topic0 = keccak256("OwnerActs(bytes4,address,bytes)");
    let mut method_topic = [0u8; 32];
    method_topic[..4].copy_from_slice(selector);
    let topic1 = B256::from(method_topic);
    let topic2 = B256::left_padding_from(input.caller.as_slice());

    // ABI-encode calldata as bytes: offset(32) + length(32) + data (padded)
    let mut log_data = Vec::with_capacity(64 + calldata.len().div_ceil(32) * 32);
    log_data.extend_from_slice(&U256::from(32).to_be_bytes::<32>()); // offset
    log_data.extend_from_slice(&U256::from(calldata.len()).to_be_bytes::<32>()); // length
    log_data.extend_from_slice(calldata);
    // Pad to 32-byte boundary
    let pad = (32 - (calldata.len() % 32)) % 32;
    log_data.extend(std::iter::repeat_n(0u8, pad));

    input.internals_mut().log(Log::new_unchecked(
        ARBOWNER_ADDRESS,
        vec![topic0, topic1, topic2],
        log_data.into(),
    ));
}

// ── AddressSet helpers ──────────────────────────────────────────────

/// Check if an address is a chain owner.
fn handle_is_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    handle_is_member(input, CHAIN_OWNER_SUBSPACE)
}

/// Get all chain owners. Returns ABI-encoded dynamic address array.
fn handle_get_all_chain_owners(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    handle_get_all_members(input, CHAIN_OWNER_SUBSPACE)
}

/// Add a chain owner. Emits ChainOwnerAdded event for ArbOS >= 60.
fn handle_add_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);

    address_set_add(input, address_set_key(CHAIN_OWNER_SUBSPACE), addr)?;

    // Emit ChainOwnerAdded event for ArbOS >= 60.
    let arbos_version = read_arbos_version(input)?;
    if arbos_version >= 60 {
        let topic0 = keccak256("ChainOwnerAdded(address)");
        let topic1 = B256::left_padding_from(addr.as_slice());
        input.internals_mut().log(Log::new_unchecked(
            ARBOWNER_ADDRESS,
            vec![topic0, topic1],
            alloy_primitives::Bytes::new(),
        ));
    }

    let gas_used = 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// Remove a chain owner. Emits ChainOwnerRemoved event for ArbOS >= 60.
fn handle_remove_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);

    let set_key = address_set_key(CHAIN_OWNER_SUBSPACE);
    if !is_member_of(input, set_key, addr)? {
        return Err(PrecompileError::other("tried to remove non-owner"));
    }
    address_set_remove(input, set_key, addr)?;

    // Emit ChainOwnerRemoved event for ArbOS >= 60.
    let arbos_version = read_arbos_version(input)?;
    if arbos_version >= 60 {
        let topic0 = keccak256("ChainOwnerRemoved(address)");
        let topic1 = B256::left_padding_from(addr.as_slice());
        input.internals_mut().log(Log::new_unchecked(
            ARBOWNER_ADDRESS,
            vec![topic0, topic1],
            alloy_primitives::Bytes::new(),
        ));
    }

    let gas_used = 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// Release surplus L1 pricer funds.
///
/// surplus = pool_balance - recognized_fees; capped by maxWeiToRelease.
/// Adds the released amount to L1FeesAvailable rather than zeroing it.
fn handle_release_l1_pricer_surplus_funds(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let max_wei = U256::from_be_slice(&data[4..36]);

    // Read pool account balance.
    let pool_balance = {
        let acct = input
            .internals_mut()
            .load_account(L1_PRICER_FUNDS_POOL_ADDRESS)
            .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
        acct.data.info.balance
    };

    // Read recognized fees (L1FeesAvailable).
    load_arbos(input)?;
    let avail_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_FEES_AVAILABLE);
    let recognized = sload_field(input, avail_slot)?;

    // Compute surplus = pool_balance - recognized.
    if pool_balance <= recognized {
        // No surplus.
        return Ok(PrecompileOutput::new(
            (SLOAD_GAS + COPY_GAS + 100).min(gas_limit),
            U256::ZERO.to_be_bytes::<32>().to_vec().into(),
        ));
    }

    let mut wei_to_transfer = pool_balance - recognized;
    if wei_to_transfer > max_wei {
        wei_to_transfer = max_wei;
    }

    // Add to L1FeesAvailable.
    let new_available = recognized + wei_to_transfer;
    sstore_field(input, avail_slot, new_available)?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + SSTORE_GAS + COPY_GAS + 100).min(gas_limit),
        wei_to_transfer.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Derive the storage key for an AddressSet at the given subspace.
fn address_set_key(subspace: &[u8]) -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, subspace)
}

/// Check if an address is a member of the AddressSet with the given set key.
fn is_member_of(
    input: &mut PrecompileInput<'_>,
    set_key: B256,
    addr: Address,
) -> Result<bool, PrecompileError> {
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let val = sload_field(input, slot)?;
    Ok(val != U256::ZERO)
}

/// Handle IS_MEMBER check for an address set.
fn handle_is_member(input: &mut PrecompileInput<'_>, subspace: &[u8]) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    let is_member = is_member_of(input, address_set_key(subspace), addr)?;
    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Handle GET_ALL_MEMBERS for an address set. Returns ABI-encoded dynamic array.
fn handle_get_all_members(input: &mut PrecompileInput<'_>, subspace: &[u8]) -> PrecompileResult {
    let gas_limit = input.gas;
    let set_key = address_set_key(subspace);
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let count: u64 = size
        .try_into()
        .map_err(|_| PrecompileError::other("invalid address set size"))?;
    const MAX_MEMBERS: u64 = 65536;
    let count = count.min(MAX_MEMBERS);

    let mut addresses = Vec::with_capacity(count as usize);
    for i in 1..=count {
        let member_slot = map_slot(set_key.as_slice(), i);
        let val = sload_field(input, member_slot)?;
        addresses.push(val);
    }

    let mut out = Vec::with_capacity(64 + 32 * addresses.len());
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for addr_val in &addresses {
        out.extend_from_slice(&addr_val.to_be_bytes::<32>());
    }

    let gas_used = (1 + count) * SLOAD_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), out.into()))
}

/// Add an address to an AddressSet. Returns true if newly added.
fn address_set_add(
    input: &mut PrecompileInput<'_>,
    set_key: B256,
    addr: Address,
) -> Result<bool, PrecompileError> {
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let existing = sload_field(input, member_slot)?;

    if existing != U256::ZERO {
        return Ok(false); // already a member
    }

    // Read size and increment.
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let size_u64: u64 = size
        .try_into()
        .map_err(|_| PrecompileError::other("invalid address set size"))?;
    let new_size = size_u64 + 1;

    // Store address at position (1 + old_size).
    let new_pos_slot = map_slot(set_key.as_slice(), new_size);
    let addr_as_u256 = U256::from_be_slice(addr.as_slice());
    sstore_field(input, new_pos_slot, addr_as_u256)?;

    // Store in byAddress: addr_hash → 1-based position.
    sstore_field(input, member_slot, U256::from(new_size))?;

    // Update size.
    sstore_field(input, size_slot, U256::from(new_size))?;

    Ok(true)
}

/// Remove an address from an AddressSet using swap-with-last.
fn address_set_remove(
    input: &mut PrecompileInput<'_>,
    set_key: B256,
    addr: Address,
) -> Result<(), PrecompileError> {
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);

    // Get the 1-based position of the address.
    let position = sload_field(input, member_slot)?;
    if position == U256::ZERO {
        return Err(PrecompileError::other("address not in set"));
    }
    let pos_u64: u64 = position
        .try_into()
        .map_err(|_| PrecompileError::other("invalid position"))?;

    // Clear the byAddress entry.
    sstore_field(input, member_slot, U256::ZERO)?;

    // Read current size.
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let size_u64: u64 = size
        .try_into()
        .map_err(|_| PrecompileError::other("invalid size"))?;

    // If not the last element, swap with last.
    if pos_u64 < size_u64 {
        let last_slot = map_slot(set_key.as_slice(), size_u64);
        let last_val = sload_field(input, last_slot)?;

        // Move last to removed position.
        let removed_slot = map_slot(set_key.as_slice(), pos_u64);
        sstore_field(input, removed_slot, last_val)?;

        // Update byAddress for the swapped address.
        let last_bytes = last_val.to_be_bytes::<32>();
        let last_hash = B256::from(last_bytes);
        let last_member_slot = map_slot_b256(by_address_key.as_slice(), &last_hash);
        sstore_field(input, last_member_slot, U256::from(pos_u64))?;
    }

    // Clear last position.
    let last_pos_slot = map_slot(set_key.as_slice(), size_u64);
    sstore_field(input, last_pos_slot, U256::ZERO)?;

    // Decrement size.
    sstore_field(input, size_slot, U256::from(size_u64 - 1))?;

    Ok(())
}

// ── Stylus param helpers ─────────────────────────────────────────────

/// Field identifiers in the packed StylusParams storage word.
///
/// Packed layout (byte offsets within a 32-byte storage word):
///   [0-1]   version (u16)
///   [2-4]   ink_price (uint24)
///   [5-8]   max_stack_depth (u32)
///   [9-10]  free_pages (u16)
///   [11-12] page_gas (u16)
///   [13-14] page_limit (u16)
///   [15]    min_init_gas (u8)
///   [16]    min_cached_init_gas (u8)
///   [17]    init_cost_scalar (u8)
///   [18]    cached_cost_scalar (u8)
///   [19-20] expiry_days (u16)
///   [21-22] keepalive_days (u16)
///   [23-24] block_cache_size (u16)
///   [25-28] max_wasm_size (u32) -- arbos >= 40
///   [29]    max_fragment_count (u8) -- arbos >= 41
enum StylusField {
    InkPrice,         // bytes 2..5 (uint24)
    MaxStackDepth,    // bytes 5..9 (u32)
    FreePages,        // bytes 9..11 (u16)
    PageGas,          // bytes 11..13 (u16)
    PageLimit,        // bytes 13..15 (u16)
    MinInitGas,       // byte 15 (u8)
    InitCostScalar,   // byte 17 (u8)
    ExpiryDays,       // bytes 19..21 (u16)
    KeepaliveDays,    // bytes 21..23 (u16)
    BlockCacheSize,   // bytes 23..25 (u16)
    MaxWasmSize,      // bytes 25..29 (u32)
    MaxFragmentCount, // byte 29 (u8)
}

impl StylusField {
    fn byte_range(&self) -> (usize, usize) {
        match self {
            Self::InkPrice => (2, 5),
            Self::MaxStackDepth => (5, 9),
            Self::FreePages => (9, 11),
            Self::PageGas => (11, 13),
            Self::PageLimit => (13, 15),
            Self::MinInitGas => (15, 16),
            Self::InitCostScalar => (17, 18),
            Self::ExpiryDays => (19, 21),
            Self::KeepaliveDays => (21, 23),
            Self::BlockCacheSize => (23, 25),
            Self::MaxWasmSize => (25, 29),
            Self::MaxFragmentCount => (29, 30),
        }
    }
}

/// Compute the storage slot for the programs params word (slot 0).
fn programs_params_slot() -> U256 {
    // Path: root → PROGRAMS_SUBSPACE → PROGRAMS_PARAMS_KEY → slot 0
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), &[0]); // PARAMS_KEY = [0]
    map_slot(params_key.as_slice(), 0)
}

/// Read the packed programs params word from storage.
fn read_stylus_params_word(input: &mut PrecompileInput<'_>) -> Result<[u8; 32], PrecompileError> {
    let slot = programs_params_slot();
    let val = sload_field(input, slot)?;
    Ok(val.to_be_bytes::<32>())
}

/// Write a modified field in the packed programs params storage word.
fn write_stylus_param(
    input: &mut PrecompileInput<'_>,
    field: StylusField,
    value: u64,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let slot = programs_params_slot();
    let mut word = read_stylus_params_word(input)?;

    let (start, end) = field.byte_range();
    let len = end - start;
    let bytes = value.to_be_bytes();
    // Copy the least significant `len` bytes from the u64.
    word[start..end].copy_from_slice(&bytes[8 - len..]);

    sstore_field(input, slot, U256::from_be_bytes(word))?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

/// Parse a u32 parameter from ABI calldata (first arg after selector).
fn read_u32_param(data: &[u8]) -> Result<u32, PrecompileError> {
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }
    let val = U256::from_be_slice(&data[4..36]);
    val.try_into()
        .map_err(|_| PrecompileError::other("value overflow"))
}

// ── Cache manager helpers ────────────────────────────────────────────

/// Derive the AddressSet key for cache managers (PROGRAMS → CACHE_MANAGERS).
fn cache_managers_set_key() -> B256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    derive_subspace_key(programs_key.as_slice(), CACHE_MANAGERS_KEY)
}

fn handle_add_cache_manager(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    address_set_add(input, cache_managers_set_key(), addr)?;
    let gas_used = 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_remove_cache_manager(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    let set_key = cache_managers_set_key();
    if !is_member_of(input, set_key, addr)? {
        return Err(PrecompileError::other("address is not a cache manager"));
    }
    address_set_remove(input, set_key, addr)?;
    let gas_used = 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// One week in seconds.
const FEATURE_ENABLE_DELAY: u64 = 7 * 24 * 60 * 60;

/// Handle setting a feature enabled-from timestamp with validation.
fn handle_set_feature_time(input: &mut PrecompileInput<'_>, time_offset: u64) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let timestamp: u64 = U256::from_be_slice(&data[4..36])
        .try_into()
        .map_err(|_| PrecompileError::other("timestamp too large"))?;

    if timestamp == 0 {
        // Disable the feature.
        sstore_field(input, root_slot(time_offset), U256::ZERO)?;
        return Ok(PrecompileOutput::new(
            (SSTORE_GAS + COPY_GAS).min(gas_limit),
            Vec::new().into(),
        ));
    }

    let stored_val = sload_field(input, root_slot(time_offset))?;
    let stored: u64 = stored_val.try_into().unwrap_or(0);
    let now = input
        .internals_mut()
        .block_timestamp()
        .try_into()
        .unwrap_or(0u64);

    // Validate timing constraints.
    if (stored > now + FEATURE_ENABLE_DELAY || stored == 0)
        && timestamp < now + FEATURE_ENABLE_DELAY
    {
        return Err(PrecompileError::other(
            "feature must be enabled at least 7 days in the future",
        ));
    }
    if stored > now && stored <= now + FEATURE_ENABLE_DELAY && timestamp < stored {
        return Err(PrecompileError::other(
            "feature cannot be updated to a time earlier than the current scheduled enable time",
        ));
    }

    sstore_field(input, root_slot(time_offset), U256::from(timestamp))?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + SSTORE_GAS + COPY_GAS).min(gas_limit),
        Vec::new().into(),
    ))
}

/// Add an address to an AddressSet, requiring that the feature is enabled.
fn handle_add_to_set_with_feature_check(
    input: &mut PrecompileInput<'_>,
    subspace: &[u8],
    time_offset: u64,
) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);

    // Check feature is enabled.
    let enabled_time_val = sload_field(input, root_slot(time_offset))?;
    let enabled_time: u64 = enabled_time_val.try_into().unwrap_or(0);
    let now: u64 = input
        .internals_mut()
        .block_timestamp()
        .try_into()
        .unwrap_or(0u64);

    if enabled_time == 0 || enabled_time > now {
        return Err(PrecompileError::other("feature is not enabled yet"));
    }

    address_set_add(input, address_set_key(subspace), addr)?;

    let gas_used = 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// Remove an address from an AddressSet.
fn handle_remove_from_set(input: &mut PrecompileInput<'_>, subspace: &[u8]) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);

    // Verify membership first.
    let set_key = address_set_key(subspace);
    if !is_member_of(input, set_key, addr)? {
        return Err(PrecompileError::other("address is not a member"));
    }

    address_set_remove(input, set_key, addr)?;

    let gas_used = 3 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

// ── Gas constraint storage helpers ───────────────────────────────────

const GC_KEY: &[u8] = b"gc";
const MGC_KEY: &[u8] = b"mgc";
const CONSTRAINT_TARGET: u64 = 0;
const CONSTRAINT_WINDOW: u64 = 1;
const CONSTRAINT_BACKLOG: u64 = 2;
const MGC_MAX_WEIGHT: u64 = 3;
const MGC_WEIGHTS_BASE: u64 = 4;
const NUM_RESOURCE_KINDS: u64 = 8;
const GAS_CONSTRAINTS_MAX_NUM: usize = 20;
const MAX_PRICING_EXPONENT_BIPS: u64 = 85_000;

/// Derive the L2 pricing sub-storage key.
fn l2_pricing_key() -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, L2_PRICING_SUBSPACE)
}

/// Derive the gas constraints vector sub-storage key.
fn gc_vector_key() -> B256 {
    derive_subspace_key(l2_pricing_key().as_slice(), GC_KEY)
}

/// Derive the multi-gas constraints vector sub-storage key.
fn mgc_vector_key() -> B256 {
    derive_subspace_key(l2_pricing_key().as_slice(), MGC_KEY)
}

/// SubStorageVector length slot (offset 0 from vector base key).
fn vector_length_slot(vector_key: B256) -> U256 {
    map_slot(vector_key.as_slice(), 0)
}

/// Sub-storage key for element at given index within a vector.
fn vector_element_key(vector_key: B256, index: u64) -> B256 {
    derive_subspace_key(vector_key.as_slice(), &index.to_be_bytes())
}

/// Storage slot for a field within a constraint element.
fn constraint_field_slot(element_key: B256, field_offset: u64) -> U256 {
    map_slot(element_key.as_slice(), field_offset)
}

/// Read ArbOS version from root state.
fn read_arbos_version(input: &mut PrecompileInput<'_>) -> Result<u64, PrecompileError> {
    let val = sload_field(input, root_slot(0))?; // VERSION_OFFSET = 0
    val.try_into()
        .map_err(|_| PrecompileError::other("invalid ArbOS version"))
}

/// Clear all constraints in a SubStorageVector of gas constraints.
fn clear_gas_constraints_vector(
    input: &mut PrecompileInput<'_>,
    vector_key: B256,
    fields_per_element: u64,
) -> Result<u64, PrecompileError> {
    let len_slot = vector_length_slot(vector_key);
    let len: u64 = sload_field(input, len_slot)?.try_into().unwrap_or(0);

    // Clear each constraint's fields.
    for i in 0..len {
        let elem_key = vector_element_key(vector_key, i);
        for f in 0..fields_per_element {
            sstore_field(input, constraint_field_slot(elem_key, f), U256::ZERO)?;
        }
    }

    // Reset length to 0.
    sstore_field(input, len_slot, U256::ZERO)?;

    Ok(len)
}

/// Clear all multi-gas constraints, including resource weights.
fn clear_multi_gas_constraints(input: &mut PrecompileInput<'_>) -> Result<u64, PrecompileError> {
    let vector_key = mgc_vector_key();
    let len_slot = vector_length_slot(vector_key);
    let len: u64 = sload_field(input, len_slot)?.try_into().unwrap_or(0);

    for i in 0..len {
        let elem_key = vector_element_key(vector_key, i);
        // Clear target, window, backlog, max_weight.
        for f in 0..4 {
            sstore_field(input, constraint_field_slot(elem_key, f), U256::ZERO)?;
        }
        // Clear resource weights.
        for r in 0..NUM_RESOURCE_KINDS {
            sstore_field(
                input,
                constraint_field_slot(elem_key, MGC_WEIGHTS_BASE + r),
                U256::ZERO,
            )?;
        }
    }

    sstore_field(input, len_slot, U256::ZERO)?;
    Ok(len)
}

// ── SetGasPricingConstraints ─────────────────────────────────────────

/// ABI: `setGasPricingConstraints(uint64[3][])`
///
/// Clears existing constraints, validates count for certain ArbOS versions,
/// then adds each constraint (target, adjustment_window, starting_backlog).
fn handle_set_gas_pricing_constraints(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    // Minimum: selector(4) + offset(32) + length(32) = 68 bytes
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    // Parse array length.
    let count: u64 = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| PrecompileError::other("array length overflow"))?;

    // Each element is 3 × 32 bytes = 96 bytes.
    let expected_len = 68 + (count as usize) * 96;
    if data.len() < expected_len {
        return crate::burn_all_revert(gas_limit);
    }

    // Clear existing constraints.
    let vector_key = gc_vector_key();
    clear_gas_constraints_vector(input, vector_key, 3)?;

    // Version check for max constraint count.
    let arbos_version = read_arbos_version(input)?;
    use arb_chainspec::arbos_version as arb_ver;
    if (arb_ver::ARBOS_VERSION_MULTI_CONSTRAINT_FIX..arb_ver::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS)
        .contains(&arbos_version)
        && (count as usize) > GAS_CONSTRAINTS_MAX_NUM
    {
        return Err(PrecompileError::other("too many constraints"));
    }

    // Add each constraint.
    let len_slot = vector_length_slot(vector_key);
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
            return Err(PrecompileError::other("invalid constraint parameters"));
        }

        // Write constraint fields.
        let elem_key = vector_element_key(vector_key, i);
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_TARGET),
            U256::from(target),
        )?;
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_WINDOW),
            U256::from(window),
        )?;
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_BACKLOG),
            U256::from(backlog),
        )?;

        // Increment vector length.
        sstore_field(input, len_slot, U256::from(i + 1))?;
    }

    let gas_used = (count * 4 + 2) * SSTORE_GAS + count * SLOAD_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

// ── SetMultiGasPricingConstraints ────────────────────────────────────

/// ABI: `setMultiGasPricingConstraints((uint64,uint64,uint64,(uint8,uint64)[])[])`
///
/// Clears existing multi-gas constraints, then adds each constraint
/// with per-resource-kind weights. Validates pricing exponents after each add.
fn handle_set_multi_gas_pricing_constraints(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    // Clear existing multi-gas constraints.
    clear_multi_gas_constraints(input)?;

    // The outer array offset.
    let _outer_offset: usize = U256::from_be_slice(&data[4..36])
        .try_into()
        .unwrap_or(0usize);
    // Array length.
    let count: u64 = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| PrecompileError::other("array length overflow"))?;

    // Parse each constraint. ABI encodes dynamic structs with offsets.
    let array_data_start = 68; // after selector + offset + length

    // Read offsets to each struct.
    let mut struct_offsets = Vec::with_capacity(count as usize);
    for i in 0..count as usize {
        let offset_pos = array_data_start + i * 32;
        if data.len() < offset_pos + 32 {
            return crate::burn_all_revert(gas_limit);
        }
        let offset: usize = U256::from_be_slice(&data[offset_pos..offset_pos + 32])
            .try_into()
            .unwrap_or(0);
        // Offset is relative to array data start.
        struct_offsets.push(array_data_start + offset);
    }

    let vector_key = mgc_vector_key();
    let len_slot = vector_length_slot(vector_key);

    for (i, &struct_start) in struct_offsets.iter().enumerate() {
        // Each struct: target(32) + window(32) + backlog(32) + resources_offset(32) = 128 min
        if data.len() < struct_start + 128 {
            return crate::burn_all_revert(gas_limit);
        }

        let target: u64 = U256::from_be_slice(&data[struct_start..struct_start + 32])
            .try_into()
            .unwrap_or(0);
        let window: u64 = U256::from_be_slice(&data[struct_start + 32..struct_start + 64])
            .try_into()
            .unwrap_or(0);
        let backlog: u64 = U256::from_be_slice(&data[struct_start + 64..struct_start + 96])
            .try_into()
            .unwrap_or(0);

        if target == 0 || window == 0 {
            return Err(PrecompileError::other("invalid constraint parameters"));
        }

        // Parse resources offset (relative to struct start).
        let resources_offset: usize =
            U256::from_be_slice(&data[struct_start + 96..struct_start + 128])
                .try_into()
                .unwrap_or(0);
        let resources_start = struct_start + resources_offset;

        if data.len() < resources_start + 32 {
            return crate::burn_all_revert(gas_limit);
        }

        let num_resources: usize =
            U256::from_be_slice(&data[resources_start..resources_start + 32])
                .try_into()
                .unwrap_or(0);

        // Parse resource weights.
        let mut weights = [0u64; 8];
        let mut max_weight = 0u64;
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

            if (resource as u64) < NUM_RESOURCE_KINDS {
                weights[resource as usize] = weight;
                if weight > max_weight {
                    max_weight = weight;
                }
            }
        }

        // Write constraint to storage.
        let elem_key = vector_element_key(vector_key, i as u64);
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_TARGET),
            U256::from(target),
        )?;
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_WINDOW),
            U256::from(window),
        )?;
        sstore_field(
            input,
            constraint_field_slot(elem_key, CONSTRAINT_BACKLOG),
            U256::from(backlog),
        )?;
        sstore_field(
            input,
            constraint_field_slot(elem_key, MGC_MAX_WEIGHT),
            U256::from(max_weight),
        )?;

        // Write resource weights.
        for (r, &weight) in weights.iter().enumerate().take(NUM_RESOURCE_KINDS as usize) {
            sstore_field(
                input,
                constraint_field_slot(elem_key, MGC_WEIGHTS_BASE + r as u64),
                U256::from(weight),
            )?;
        }

        // Increment vector length.
        sstore_field(input, len_slot, U256::from(i as u64 + 1))?;

        // Validate exponents after each constraint.
        validate_multi_gas_exponents(input, vector_key, i as u64 + 1)?;
    }

    let gas_used = (count * 16 + 2) * SSTORE_GAS + (count * 12 + 2) * SLOAD_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// Validate that no pricing exponent exceeds the maximum.
fn validate_multi_gas_exponents(
    input: &mut PrecompileInput<'_>,
    vector_key: B256,
    count: u64,
) -> Result<(), PrecompileError> {
    let mut exponents = [0u64; 8];

    for i in 0..count {
        let elem_key = vector_element_key(vector_key, i);
        let target: u64 = sload_field(input, constraint_field_slot(elem_key, CONSTRAINT_TARGET))?
            .try_into()
            .unwrap_or(0);
        let backlog: u64 = sload_field(input, constraint_field_slot(elem_key, CONSTRAINT_BACKLOG))?
            .try_into()
            .unwrap_or(0);

        if backlog == 0 {
            continue;
        }

        let window: u64 = sload_field(input, constraint_field_slot(elem_key, CONSTRAINT_WINDOW))?
            .try_into()
            .unwrap_or(0);
        let max_weight: u64 = sload_field(input, constraint_field_slot(elem_key, MGC_MAX_WEIGHT))?
            .try_into()
            .unwrap_or(0);

        if max_weight == 0 || target == 0 || window == 0 {
            continue;
        }

        // divisor = window * target * max_weight (in bips).
        let divisor = (window as u128)
            .saturating_mul(target as u128)
            .saturating_mul(max_weight as u128);
        let divisor_bips = divisor.saturating_mul(10000);

        for (r, exponent) in exponents
            .iter_mut()
            .enumerate()
            .take(NUM_RESOURCE_KINDS as usize)
        {
            let weight: u64 = sload_field(
                input,
                constraint_field_slot(elem_key, MGC_WEIGHTS_BASE + r as u64),
            )?
            .try_into()
            .unwrap_or(0);

            if weight == 0 {
                continue;
            }

            let dividend = (backlog as u128).saturating_mul(weight as u128) * 10000;
            let exp = if divisor_bips > 0 {
                (dividend / divisor_bips) as u64
            } else {
                0
            };
            *exponent = exponent.saturating_add(exp);
        }
    }

    for &exp in &exponents {
        if exp > MAX_PRICING_EXPONENT_BIPS {
            return Err(PrecompileError::other("pricing exponent exceeds maximum"));
        }
    }

    Ok(())
}

// ── SetChainConfig ───────────────────────────────────────────────────

/// ABI: `setChainConfig(bytes)`
///
/// Stores the serialized chain config in StorageBackedBytes format.
fn handle_set_chain_config(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;

    // ABI: offset(32) + length(32) + data.
    let bytes_len: usize = U256::from_be_slice(&data[36..68])
        .try_into()
        .map_err(|_| PrecompileError::other("bytes length overflow"))?;

    if data.len() < 68 + bytes_len {
        return crate::burn_all_revert(gas_limit);
    }
    let config_bytes = &data[68..68 + bytes_len];

    // Chain config sub-storage key.
    let cc_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_CONFIG_SUBSPACE);

    // Clear existing bytes (StorageBackedBytes: slot 0 = length, slots 1..N = data).
    let old_len: u64 = sload_field(input, map_slot(cc_key.as_slice(), 0))?
        .try_into()
        .unwrap_or(0);
    let old_slots = old_len.div_ceil(32);
    for s in 1..=old_slots {
        sstore_field(input, map_slot(cc_key.as_slice(), s), U256::ZERO)?;
    }

    // Write new length.
    sstore_field(
        input,
        map_slot(cc_key.as_slice(), 0),
        U256::from(bytes_len as u64),
    )?;

    // Write data in 32-byte chunks.
    let mut remaining = config_bytes;
    let mut offset = 1u64;
    while remaining.len() >= 32 {
        let mut slot = [0u8; 32];
        slot.copy_from_slice(&remaining[..32]);
        sstore_field(
            input,
            map_slot(cc_key.as_slice(), offset),
            U256::from_be_bytes(slot),
        )?;
        remaining = &remaining[32..];
        offset += 1;
    }
    if !remaining.is_empty() {
        let mut slot = [0u8; 32];
        slot[..remaining.len()].copy_from_slice(remaining);
        sstore_field(
            input,
            map_slot(cc_key.as_slice(), offset),
            U256::from_be_bytes(slot),
        )?;
    }

    let new_slots = (bytes_len as u64).div_ceil(32);
    let total_stores = old_slots + 1 + new_slots; // clear + length + data
    let gas_used = total_stores * SSTORE_GAS + SLOAD_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

/// Set the calldata price increase feature flag.
/// Reads a bool from calldata and sets bit 0 of the features bitmask.
fn handle_set_calldata_price_increase(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }
    let gas_limit = input.gas;
    let enabled = U256::from_be_slice(&data[4..36]) != U256::ZERO;

    load_arbos(input)?;

    let features_key = derive_subspace_key(ROOT_STORAGE_KEY, FEATURES_SUBSPACE);
    let features_slot = map_slot(features_key.as_slice(), 0);
    let current = sload_field(input, features_slot)?;

    let updated = if enabled {
        current | U256::from(1)
    } else {
        current & !(U256::from(1))
    };
    sstore_field(input, features_slot, updated)?;

    let gas_used = SLOAD_GAS + SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}
