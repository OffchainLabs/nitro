//! ArbOS genesis state initialization.
//!
//! Initializes the ArbOS system state in the database when the chain boots.
//! This runs when the first message (Kind=11, Initialize) is received from
//! the Nitro consensus sidecar.

use alloy_primitives::{Address, Bytes, B256, U256, address};
use revm::database::State;
use revm::Database;
use tracing::info;

use arb_storage::{Storage, StorageBackedBigUint, set_account_code};
use arbos::arbos_state::ArbosState;
use arbos::arbos_types::ParsedInitMessage;
use arbos::burn::SystemBurner;
use arbos::l1_pricing;
use arbos::l2_pricing;

/// Well-known precompile address range: 0x64 through 0xff.
const PRECOMPILE_FIRST: u8 = 0x64;
const PRECOMPILE_LAST: u8 = 0xff;

/// The initial ArbOS version for Arbitrum Sepolia genesis.
/// The upgrade_arbos_version path handles stepping through all intermediate versions.
pub const INITIAL_ARBOS_VERSION: u64 = 32;

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
pub fn initialize_arbos_state<D: Database>(
    state: &mut State<D>,
    init_msg: &ParsedInitMessage,
    chain_id: u64,
    target_arbos_version: u64,
    chain_owner: Address,
) -> Result<(), String> {
    let state_ptr: *mut State<D> = state as *mut State<D>;

    // Check if already initialized (version != 0 means state exists).
    let backing = Storage::new(state_ptr, B256::ZERO);
    if backing.get_uint64_by_uint64(0).unwrap_or(0) != 0 {
        return Err("ArbOS state already initialized".into());
    }

    info!(
        target: "arb::genesis",
        chain_id,
        target_arbos_version,
        initial_l1_base_fee = %init_msg.initial_l1_base_fee,
        "Initializing ArbOS state"
    );

    // 1. Set version to 1 (base version before upgrades).
    backing
        .set_by_uint64(0, B256::from(U256::from(1u64)))
        .map_err(|_| "failed to set initial version")?;

    // 2. Set chain ID.
    StorageBackedBigUint::new(state_ptr, B256::ZERO, 4)
        .set(U256::from(chain_id))
        .map_err(|_| "failed to set chain ID")?;

    // 3. Install precompile code markers (0x64 through 0xff).
    for addr_byte in PRECOMPILE_FIRST..=PRECOMPILE_LAST {
        let mut addr_bytes = [0u8; 20];
        addr_bytes[19] = addr_byte;
        let addr = Address::new(addr_bytes);
        set_account_code(state, addr, Bytes::from_static(&[0xFE]));
    }

    // 4. Initialize L1 pricing state.
    let l1_sto = backing.open_sub_storage(&[0]); // L1_PRICING_SUBSPACE
    l1_pricing::L1PricingState::initialize(
        &l1_sto,
        Address::ZERO, // rewards recipient (initially zero)
        init_msg.initial_l1_base_fee,
    );

    // 5. Initialize L2 pricing state.
    let l2_sto = backing.open_sub_storage(&[1]); // L2_PRICING_SUBSPACE
    l2_pricing::L2PricingState::initialize(&l2_sto);

    // 6. Initialize retryable state.
    let ret_sto = backing.open_sub_storage(&[2]); // RETRYABLES_SUBSPACE
    arbos::retryables::RetryableState::initialize(&ret_sto)
        .map_err(|_| "failed to initialize retryable state")?;

    // 7. Initialize address table (no-op but call for consistency).
    let at_sto = backing.open_sub_storage(&[3]); // ADDRESS_TABLE_SUBSPACE
    arbos::address_table::initialize_address_table(&at_sto);

    // 8. Initialize chain owners.
    let co_sto = backing.open_sub_storage(&[4]); // CHAIN_OWNER_SUBSPACE
    arbos::address_set::initialize_address_set(&co_sto)
        .map_err(|_| "failed to initialize chain owners")?;

    // 9. Initialize merkle accumulator.
    let ma_sto = backing.open_sub_storage(&[5]); // SEND_MERKLE_SUBSPACE
    arbos::merkle_accumulator::initialize_merkle_accumulator(&ma_sto);

    // 10. Initialize blockhashes.
    let bh_sto = backing.open_sub_storage(&[6]); // BLOCKHASHES_SUBSPACE
    arbos::blockhash::initialize_blockhashes(&bh_sto);

    // 11. Initialize features.
    let _feat_sto = backing.open_sub_storage(&[9]); // FEATURES_SUBSPACE

    // Now open ArbOS state and run the upgrade path from v1 to target version.
    // The open() method reads version from storage (we set it to 1 above).
    let mut arb_state = ArbosState::open(state_ptr, SystemBurner::new(None, false))
        .map_err(|_| "failed to open ArbOS state after initial setup")?;

    // Add chain owner.
    if chain_owner != Address::ZERO {
        arb_state
            .chain_owners
            .add(chain_owner)
            .map_err(|_| "failed to add chain owner")?;
    }

    // Run version upgrade from 1 to target (first_time=true).
    if target_arbos_version > 1 {
        arb_state
            .upgrade_arbos_version(target_arbos_version, true)
            .map_err(|_| format!("failed to upgrade ArbOS to version {target_arbos_version}"))?;
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
    let state_ptr: *mut State<D> = state as *mut State<D>;
    let backing = Storage::new(state_ptr, B256::ZERO);
    backing.get_uint64_by_uint64(0).unwrap_or(0) != 0
}
