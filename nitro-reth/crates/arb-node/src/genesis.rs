//! ArbOS genesis state initialization.
//!
//! Initializes the ArbOS system state in the database when the chain boots.
//! Runs when the first message (Kind=11, Initialize) is received from the
//! consensus sidecar.

use alloy_primitives::{address, Address, Bytes, B256, U256};
use revm::{database::State, Database};
use tracing::info;

use arb_storage::{
    layout::{
        ADDRESS_TABLE_SUBSPACE, BLOCKHASHES_SUBSPACE, CHAIN_CONFIG_SUBSPACE, CHAIN_OWNER_SUBSPACE,
        FEATURES_SUBSPACE, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE, RETRYABLES_SUBSPACE,
        SEND_MERKLE_SUBSPACE,
    },
    set_account_code, set_account_nonce, Storage, StorageBackedBigUint, StorageBackedBytes,
    ARBOS_STATE_ADDRESS,
};
use arbos::{
    arbos_state::ArbosState, arbos_types::ParsedInitMessage, burn::SystemBurner, l1_pricing,
    l2_pricing,
};

use crate::error::GenesisError;

/// Precompile addresses that exist at genesis (version 0).
/// Only these get the `[0xFE]` invalid code marker at init time.
/// Later precompiles (ArbWasm, ArbWasmCache, etc.) get code when their
/// ArbOS version is reached during the upgrade path.
const GENESIS_PRECOMPILE_ADDRESSES: [Address; 14] = [
    address!("0000000000000000000000000000000000000064"), // ArbSys
    address!("0000000000000000000000000000000000000065"), // ArbInfo
    address!("0000000000000000000000000000000000000066"), // ArbAddressTable
    address!("0000000000000000000000000000000000000067"), // ArbBLS
    address!("0000000000000000000000000000000000000068"), // ArbFunctionTable
    address!("0000000000000000000000000000000000000069"), // ArbosTest
    address!("000000000000000000000000000000000000006b"), // ArbOwnerPublic
    address!("000000000000000000000000000000000000006c"), // ArbGasInfo
    address!("000000000000000000000000000000000000006d"), // ArbAggregator
    address!("000000000000000000000000000000000000006e"), // ArbRetryableTx
    address!("000000000000000000000000000000000000006f"), // ArbStatistics
    address!("0000000000000000000000000000000000000070"), // ArbOwner
    address!("00000000000000000000000000000000000000ff"), // ArbDebug
    address!("00000000000000000000000000000000000a4b05"), // ArbosActs
];

/// The initial ArbOS version for Arbitrum Sepolia genesis.
/// The upgrade_arbos_version path handles stepping through all intermediate versions.
pub const INITIAL_ARBOS_VERSION: u64 = 10;

/// Default chain owner for Arbitrum Sepolia.
pub const DEFAULT_CHAIN_OWNER: Address = address!("0000000000000000000000000000000000000000");

/// Initialize ArbOS state in a freshly created database.
///
/// This sets up:
/// - ArbOS version (set to 1, then upgrade to target version)
/// - All precompile accounts with `[0xFE]` invalid code marker
/// - L1 pricing state (initial base fee, batch poster table)
/// - L2 pricing state (base fee, gas pool, speed limit)
/// - Retryable state, address table, merkle accumulator, blockhashes
/// - Chain owner and chain config
///
/// The `init_msg` comes from parsing the L1 Initialize message (Kind=11).
#[derive(Debug, Clone, Copy, Default)]
pub struct ArbOSInit {
    pub native_token_supply_management_enabled: bool,
    pub transaction_filtering_enabled: bool,
}

pub fn initialize_arbos_state<D: Database>(
    state: &mut State<D>,
    init_msg: &ParsedInitMessage,
    chain_id: u64,
    target_arbos_version: u64,
    chain_owner: Address,
    arbos_init: ArbOSInit,
) -> Result<(), GenesisError> {
    let backing = Storage::new(state, B256::ZERO);
    if backing.get_uint64_by_uint64(0).unwrap_or(0) != 0 {
        return Err(GenesisError::AlreadyInitialized);
    }

    info!(
        target: "arb::genesis",
        chain_id,
        target_arbos_version,
        initial_l1_base_fee = %init_msg.initial_l1_base_fee,
        "Initializing ArbOS state"
    );

    // SAFETY: genesis runs single-threaded; no two state_mut borrows are live
    // concurrently. `backing` is the only live Storage handle.
    set_account_nonce(unsafe { backing.state_mut() }, ARBOS_STATE_ADDRESS, 1);

    // 1. Set version to 1 (base version before upgrades).
    backing
        .set_by_uint64(0, B256::from(U256::from(1u64)))
        .map_err(|source| GenesisError::StorageWrite {
            what: "initial version",
            source,
        })?;

    // 2. Set chain ID.
    // SAFETY: see initial state_mut() comment; no overlapping Storage handles.
    StorageBackedBigUint::new(B256::ZERO, 4)
        .set(unsafe { backing.state_mut() }, U256::from(chain_id))
        .map_err(|source| GenesisError::StorageWrite {
            what: "chain id",
            source,
        })?;

    // 3. Install precompile code markers for version-0 precompiles only.
    for addr in &GENESIS_PRECOMPILE_ADDRESSES {
        // SAFETY: see initial state_mut() comment.
        set_account_code(
            unsafe { backing.state_mut() },
            *addr,
            Bytes::from_static(&[0xFE]),
        );
    }

    // 3b. Set network fee account (chain owner for version >= 2).
    if target_arbos_version >= 2 {
        let mut hash = B256::ZERO;
        hash[12..32].copy_from_slice(chain_owner.as_slice());
        backing
            .set_by_uint64(3, hash)
            .map_err(|source| GenesisError::StorageWrite {
                what: "network fee account",
                source,
            })?;
    }

    // 3c. Store serialized chain config.
    if !init_msg.serialized_chain_config.is_empty() {
        let cc_sto = backing.open_sub_storage(CHAIN_CONFIG_SUBSPACE);
        let cc_bytes = StorageBackedBytes::new(cc_sto.base_key());
        // SAFETY: see initial state_mut() comment.
        cc_bytes
            .set(
                unsafe { backing.state_mut() },
                &init_msg.serialized_chain_config,
            )
            .map_err(|source| GenesisError::StorageWrite {
                what: "chain config",
                source,
            })?;
    }

    // 4. Initialize L1 pricing state.
    let l1_sto = backing.open_sub_storage(L1_PRICING_SUBSPACE);
    let rewards_recipient = if target_arbos_version >= 2 {
        chain_owner
    } else {
        Address::ZERO
    };
    // SAFETY: see initial state_mut() comment.
    l1_pricing::L1PricingState::initialize(
        &l1_sto,
        unsafe { backing.state_mut() },
        rewards_recipient,
        init_msg.initial_l1_base_fee,
    )
    .map_err(|e| GenesisError::InitSubsystem {
        subsystem: "L1 pricing",
        source: e.into(),
    })?;

    // 5. Initialize L2 pricing state.
    let l2_sto = backing.open_sub_storage(L2_PRICING_SUBSPACE);
    // SAFETY: see initial state_mut() comment.
    l2_pricing::L2PricingState::initialize(&l2_sto, unsafe { backing.state_mut() }).map_err(
        |e| GenesisError::InitSubsystem {
            subsystem: "L2 pricing",
            source: e.into(),
        },
    )?;

    // 6. Initialize retryable state.
    let ret_sto = backing.open_sub_storage(RETRYABLES_SUBSPACE);
    arbos::retryables::RetryableState::initialize(&ret_sto).map_err(|e| {
        GenesisError::InitSubsystem {
            subsystem: "retryables",
            source: e.into(),
        }
    })?;

    // 7. Initialize address table (no-op but call for consistency).
    let at_sto = backing.open_sub_storage(ADDRESS_TABLE_SUBSPACE);
    arbos::address_table::initialize_address_table(&at_sto);

    // 8. Initialize chain owners.
    let co_sto = backing.open_sub_storage(CHAIN_OWNER_SUBSPACE);
    arbos::address_set::initialize_address_set(&co_sto).map_err(|e| {
        GenesisError::InitSubsystem {
            subsystem: "chain owners",
            source: e.into(),
        }
    })?;

    // 9. Initialize merkle accumulator.
    let ma_sto = backing.open_sub_storage(SEND_MERKLE_SUBSPACE);
    arbos::merkle_accumulator::initialize_merkle_accumulator(&ma_sto);

    // 10. Initialize blockhashes.
    let bh_sto = backing.open_sub_storage(BLOCKHASHES_SUBSPACE);
    arbos::blockhash::initialize_blockhashes(&bh_sto);

    // 11. Initialize features.
    let _feat_sto = backing.open_sub_storage(FEATURES_SUBSPACE);

    // Open after persisting `version = 1` above. A failure here means the
    // freshly written version word is unreadable, which is unrecoverable
    // during genesis bring-up.
    // SAFETY: see initial state_mut() comment.
    let mut arb_state = ArbosState::open(
        unsafe { backing.state_mut() },
        SystemBurner::new(None, false),
    )
    .expect("open ArbOS state after genesis initial setup");

    // SAFETY: see initial state_mut() comment.
    arb_state
        .chain_owners
        .add(unsafe { backing.state_mut() }, chain_owner)
        .map_err(|e| GenesisError::InitSubsystem {
            subsystem: "chain owner",
            source: e.into(),
        })?;

    if arbos_init.native_token_supply_management_enabled {
        // SAFETY: see initial state_mut() comment.
        arb_state
            .set_native_token_management_from_time(unsafe { backing.state_mut() }, 1)
            .map_err(|source| GenesisError::InitSubsystem {
                subsystem: "native token management",
                source,
            })?;
    }
    if arbos_init.transaction_filtering_enabled {
        // SAFETY: see initial state_mut() comment.
        arb_state
            .set_transaction_filtering_from_time(unsafe { backing.state_mut() }, 1)
            .map_err(|source| GenesisError::InitSubsystem {
                subsystem: "transaction filtering",
                source,
            })?;
    }

    if target_arbos_version > 1 {
        // SAFETY: see initial state_mut() comment.
        arb_state
            .upgrade_arbos_version(unsafe { backing.state_mut() }, target_arbos_version, true)
            .map_err(|source| GenesisError::Upgrade {
                target: target_arbos_version,
                source,
            })?;
    }

    info!(
        target: "arb::genesis",
        final_version = arb_state.arbos_version(),
        "ArbOS state initialized"
    );

    Ok(())
}

/// Check if ArbOS state is already initialized in the given state database.
pub fn is_arbos_initialized<D: Database>(state: &mut State<D>) -> bool {
    let backing = Storage::new(state, B256::ZERO);
    backing.get_uint64_by_uint64(0).unwrap_or(0) != 0
}
