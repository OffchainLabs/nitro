use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
    CHAIN_OWNER_SUBSPACE, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbOwner precompile address (0x70).
pub const ARBOWNER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x70,
]);

// ── Selectors ────────────────────────────────────────────────────────

// Getters (also on ArbOwner in Go, though most are on ArbOwnerPublic)
const GET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0x3e, 0x7a, 0x47, 0xb1];
const GET_INFRA_FEE_ACCOUNT: [u8; 4] = [0x74, 0x33, 0x16, 0x04];

// Setters — ArbOS root state
const SET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0xe1, 0xa3, 0x5b, 0x12];
const SET_INFRA_FEE_ACCOUNT: [u8; 4] = [0x0b, 0x6c, 0xf6, 0x99];
const SCHEDULE_ARBOS_UPGRADE: [u8; 4] = [0x1f, 0x87, 0x0b, 0xd3];
const SET_BROTLI_COMPRESSION_LEVEL: [u8; 4] = [0x86, 0x47, 0x23, 0x97];
const SET_CHAIN_CONFIG: [u8; 4] = [0xf5, 0xb7, 0x78, 0x63];

// Setters — L2 pricing
const SET_SPEED_LIMIT: [u8; 4] = [0x2e, 0x09, 0xca, 0x2e];
const SET_L2_BASE_FEE: [u8; 4] = [0x72, 0xbc, 0x8c, 0x42];
const SET_MINIMUM_L2_BASE_FEE: [u8; 4] = [0xa7, 0x47, 0x14, 0x0c];
const SET_MAX_BLOCK_GAS_LIMIT: [u8; 4] = [0x20, 0x2c, 0xbf, 0xbd];
const SET_MAX_TX_GAS_LIMIT: [u8; 4] = [0xa3, 0xb1, 0xb3, 0x1d];
const SET_L2_GAS_PRICING_INERTIA: [u8; 4] = [0x08, 0x88, 0x56, 0x1e];
const SET_L2_GAS_BACKLOG_TOLERANCE: [u8; 4] = [0x1e, 0xda, 0xbd, 0xa6];
const SET_GAS_BACKLOG: [u8; 4] = [0x50, 0x52, 0x48, 0x93];
const SET_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0xea, 0xe0, 0x29, 0x95];
const SET_MULTI_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0x9c, 0x04, 0x2d, 0x8e];

// Setters — L1 pricing
const SET_L1_PRICING_EQUILIBRATION_UNITS: [u8; 4] = [0x69, 0x2c, 0xeb, 0x1e];
const SET_L1_PRICING_INERTIA: [u8; 4] = [0x77, 0x6d, 0xbb, 0x4e];
const SET_L1_PRICING_REWARD_RECIPIENT: [u8; 4] = [0xca, 0x27, 0x9e, 0x4e];
const SET_L1_PRICING_REWARD_RATE: [u8; 4] = [0xee, 0x65, 0x86, 0xc6];
const SET_L1_PRICE_PER_UNIT: [u8; 4] = [0x63, 0xbe, 0x3f, 0x93];
const SET_PARENT_GAS_FLOOR_PER_TOKEN: [u8; 4] = [0x07, 0x71, 0xbb, 0xc7];
const SET_PER_BATCH_GAS_CHARGE: [u8; 4] = [0x8f, 0x69, 0xb8, 0x12];
const SET_AMORTIZED_COST_CAP_BIPS: [u8; 4] = [0xa4, 0xb8, 0xdb, 0x1e];
const RELEASE_L1_PRICER_SURPLUS_FUNDS: [u8; 4] = [0xbf, 0xc5, 0x21, 0xee];
const SET_L1_BASEFEE_ESTIMATE_INERTIA: [u8; 4] = [0x11, 0xc4, 0x8a, 0x7e];

// Setters — Stylus/Wasm
const SET_INK_PRICE: [u8; 4] = [0x8a, 0x0c, 0x4b, 0x6d];
const SET_WASM_MAX_STACK_DEPTH: [u8; 4] = [0xf2, 0x41, 0x05, 0xca];
const SET_WASM_FREE_PAGES: [u8; 4] = [0x53, 0x09, 0xac, 0xc8];
const SET_WASM_PAGE_GAS: [u8; 4] = [0x82, 0x0b, 0x2b, 0x3d];
const SET_WASM_PAGE_LIMIT: [u8; 4] = [0x30, 0xc3, 0xb8, 0x41];
const SET_WASM_MIN_INIT_GAS: [u8; 4] = [0xd2, 0x56, 0x91, 0x32];
const SET_WASM_INIT_COST_SCALAR: [u8; 4] = [0x7a, 0xaf, 0x8c, 0xa6];
const SET_WASM_EXPIRY_DAYS: [u8; 4] = [0xd9, 0x13, 0xea, 0x35];
const SET_WASM_KEEPALIVE_DAYS: [u8; 4] = [0x15, 0x8c, 0x34, 0x18];
const SET_WASM_BLOCK_CACHE_SIZE: [u8; 4] = [0xce, 0x6e, 0x7e, 0x24];
const SET_WASM_MAX_SIZE: [u8; 4] = [0x67, 0x00, 0xbb, 0x59];
const ADD_WASM_CACHE_MANAGER: [u8; 4] = [0x48, 0x28, 0x2e, 0xaf];
const REMOVE_WASM_CACHE_MANAGER: [u8; 4] = [0x1e, 0xc8, 0xd5, 0x8e];
const SET_MAX_STYLUS_CONTRACT_FRAGMENTS: [u8; 4] = [0x79, 0xaf, 0xf2, 0x99];
const SET_CALLDATA_PRICE_INCREASE: [u8; 4] = [0x03, 0x27, 0x40, 0x3c];

// Transaction filtering / native token
const ADD_TRANSACTION_FILTERER: [u8; 4] = [0x84, 0x36, 0x3d, 0xbf];
const REMOVE_TRANSACTION_FILTERER: [u8; 4] = [0xd8, 0x60, 0xf6, 0xc5];
const GET_ALL_TRANSACTION_FILTERERS: [u8; 4] = [0x3d, 0xbb, 0x43, 0x98];
const IS_TRANSACTION_FILTERER: [u8; 4] = [0xa5, 0x3f, 0xef, 0x64];
const SET_TRANSACTION_FILTERING_FROM: [u8; 4] = [0x08, 0x36, 0x96, 0x1e];
const SET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0x4a, 0xc0, 0xa0, 0x45];
const GET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0x8b, 0x00, 0x16, 0x72];
const SET_NATIVE_TOKEN_MANAGEMENT_FROM: [u8; 4] = [0x1b, 0x25, 0x67, 0xaa];
const ADD_NATIVE_TOKEN_OWNER: [u8; 4] = [0xc2, 0x5d, 0xfe, 0xbb];
const REMOVE_NATIVE_TOKEN_OWNER: [u8; 4] = [0x52, 0x2e, 0xf9, 0xad];
const GET_ALL_NATIVE_TOKEN_OWNERS: [u8; 4] = [0xf5, 0xc8, 0x16, 0x7a];
const IS_NATIVE_TOKEN_OWNER: [u8; 4] = [0x40, 0xb6, 0x62, 0x08];

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

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbowner_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbowner"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    // Verify the caller is a chain owner.
    verify_owner(&mut input)?;

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        // ── Getters ──────────────────────────────────────────────
        GET_NETWORK_FEE_ACCOUNT => read_root_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        GET_INFRA_FEE_ACCOUNT => read_root_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET),
        GET_FILTERED_FUNDS_RECIPIENT => {
            // Stored in ArbOS state (offset for filtered funds recipient).
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        GET_ALL_TRANSACTION_FILTERERS | GET_ALL_NATIVE_TOKEN_OWNERS => {
            // Return empty dynamic array.
            let gas_cost = COPY_GAS.min(input.gas);
            let mut out = Vec::with_capacity(64);
            out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
            out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
            Ok(PrecompileOutput::new(gas_cost, out.into()))
        }
        IS_TRANSACTION_FILTERER | IS_NATIVE_TOKEN_OWNER => {
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }

        // ── Root state setters ───────────────────────────────────
        SET_NETWORK_FEE_ACCOUNT => write_root_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        SET_INFRA_FEE_ACCOUNT => write_root_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET),
        SET_BROTLI_COMPRESSION_LEVEL => {
            write_root_field(&mut input, BROTLI_COMPRESSION_LEVEL_OFFSET)
        }
        SCHEDULE_ARBOS_UPGRADE => handle_schedule_upgrade(&mut input),

        // ── L2 pricing setters ───────────────────────────────────
        SET_SPEED_LIMIT => write_l2_field(&mut input, L2_SPEED_LIMIT),
        SET_L2_BASE_FEE => write_l2_field(&mut input, L2_BASE_FEE),
        SET_MINIMUM_L2_BASE_FEE => write_l2_field(&mut input, L2_MIN_BASE_FEE),
        SET_MAX_BLOCK_GAS_LIMIT => write_l2_field(&mut input, L2_PER_BLOCK_GAS_LIMIT),
        SET_MAX_TX_GAS_LIMIT => write_l2_field(&mut input, L2_PER_TX_GAS_LIMIT),
        SET_L2_GAS_PRICING_INERTIA => write_l2_field(&mut input, L2_PRICING_INERTIA),
        SET_L2_GAS_BACKLOG_TOLERANCE => write_l2_field(&mut input, L2_BACKLOG_TOLERANCE),
        SET_GAS_BACKLOG => write_l2_field(&mut input, L2_GAS_BACKLOG),

        // ── L1 pricing setters ───────────────────────────────────
        SET_L1_PRICING_EQUILIBRATION_UNITS => write_l1_field(&mut input, L1_EQUILIBRATION_UNITS),
        SET_L1_PRICING_INERTIA => write_l1_field(&mut input, L1_INERTIA),
        SET_L1_PRICING_REWARD_RECIPIENT => write_l1_field(&mut input, L1_PAY_REWARDS_TO),
        SET_L1_PRICING_REWARD_RATE => write_l1_field(&mut input, L1_PER_UNIT_REWARD),
        SET_L1_PRICE_PER_UNIT => write_l1_field(&mut input, L1_PRICE_PER_UNIT),
        SET_PARENT_GAS_FLOOR_PER_TOKEN => write_l1_field(&mut input, L1_GAS_FLOOR_PER_TOKEN),
        SET_PER_BATCH_GAS_CHARGE => write_l1_field(&mut input, L1_PER_BATCH_GAS_COST),
        SET_AMORTIZED_COST_CAP_BIPS => write_l1_field(&mut input, L1_AMORTIZED_COST_CAP_BIPS),
        SET_L1_BASEFEE_ESTIMATE_INERTIA => write_l1_field(&mut input, L1_INERTIA),
        RELEASE_L1_PRICER_SURPLUS_FUNDS => {
            // Release L1 pricer surplus — reads available then zeros it.
            let gas_limit = input.gas;
            load_arbos(&mut input)?;
            let avail_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_FEES_AVAILABLE);
            let available = sload_field(&mut input, avail_slot)?;
            sstore_field(&mut input, avail_slot, U256::ZERO)?;
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + SSTORE_GAS + COPY_GAS).min(gas_limit),
                available.to_be_bytes::<32>().to_vec().into(),
            ))
        }

        // ── Stylus/Wasm (stubs — return success) ────────────────
        SET_INK_PRICE
        | SET_WASM_MAX_STACK_DEPTH
        | SET_WASM_FREE_PAGES
        | SET_WASM_PAGE_GAS
        | SET_WASM_PAGE_LIMIT
        | SET_WASM_MIN_INIT_GAS
        | SET_WASM_INIT_COST_SCALAR
        | SET_WASM_EXPIRY_DAYS
        | SET_WASM_KEEPALIVE_DAYS
        | SET_WASM_BLOCK_CACHE_SIZE
        | SET_WASM_MAX_SIZE
        | ADD_WASM_CACHE_MANAGER
        | REMOVE_WASM_CACHE_MANAGER
        | SET_MAX_STYLUS_CONTRACT_FRAGMENTS
        | SET_CALLDATA_PRICE_INCREASE => {
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, Vec::new().into()))
        }

        // ── Transaction filtering / native token (stubs) ─────────
        ADD_TRANSACTION_FILTERER
        | REMOVE_TRANSACTION_FILTERER
        | SET_TRANSACTION_FILTERING_FROM
        | SET_FILTERED_FUNDS_RECIPIENT
        | SET_NATIVE_TOKEN_MANAGEMENT_FROM
        | ADD_NATIVE_TOKEN_OWNER
        | REMOVE_NATIVE_TOKEN_OWNER => {
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, Vec::new().into()))
        }

        // ── Multi-gas (stubs) ────────────────────────────────────
        SET_GAS_PRICING_CONSTRAINTS | SET_MULTI_GAS_PRICING_CONSTRAINTS | SET_CHAIN_CONFIG => {
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, Vec::new().into()))
        }

        _ => Err(PrecompileError::other("unknown ArbOwner selector")),
    }
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
    if value == U256::ZERO {
        return Err(PrecompileError::other("ArbOwner: caller is not a chain owner"));
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
        return Err(PrecompileError::other("input too short"));
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
        return Err(PrecompileError::other("input too short"));
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
        return Err(PrecompileError::other("input too short"));
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
        return Err(PrecompileError::other("input too short"));
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
