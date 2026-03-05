pub mod initialize;

use alloy_primitives::{Address, Bytes, B256, U256};
use revm::Database;

use arb_primitives::arbos_versions::{
    HISTORY_STORAGE_ADDRESS, HISTORY_STORAGE_CODE_ARBITRUM, PRECOMPILE_MIN_ARBOS_VERSIONS,
};
use arb_storage::{
    Storage, StorageBackedAddress, StorageBackedBigUint, StorageBackedBytes, StorageBackedUint64,
    get_account_balance, set_account_code, set_account_nonce, FILTERED_TX_STATE_ADDRESS,
};

use crate::address_set::{self, AddressSet};
use crate::address_table::{self, AddressTable};
use crate::blockhash::{self, Blockhashes};
use crate::burn::Burner;
use crate::features::{self, Features};
use crate::filtered_transactions::FilteredTransactionsState;
use crate::l1_pricing::{self, L1PricingState};
use crate::l2_pricing::{self, L2PricingState};
use crate::merkle_accumulator::{self, MerkleAccumulator};
use crate::programs::Programs;
use crate::retryables::RetryableState;

// Storage offsets for root-level ArbOS state values.
const VERSION_OFFSET: u64 = 0;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const CHAIN_ID_OFFSET: u64 = 4;
const GENESIS_BLOCK_NUM_OFFSET: u64 = 5;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET: u64 = 8;
const TRANSACTION_FILTERING_ENABLED_FROM_TIME_OFFSET: u64 = 9;
const FILTERED_FUNDS_RECIPIENT_OFFSET: u64 = 10;

// Subspace IDs for partitioned storage.
const L1_PRICING_SUBSPACE: &[u8] = &[0];
const L2_PRICING_SUBSPACE: &[u8] = &[1];
const RETRYABLES_SUBSPACE: &[u8] = &[2];
const ADDRESS_TABLE_SUBSPACE: &[u8] = &[3];
const CHAIN_OWNER_SUBSPACE: &[u8] = &[4];
const SEND_MERKLE_SUBSPACE: &[u8] = &[5];
const BLOCKHASHES_SUBSPACE: &[u8] = &[6];
const CHAIN_CONFIG_SUBSPACE: &[u8] = &[7];
const PROGRAMS_SUBSPACE: &[u8] = &[8];
const FEATURES_SUBSPACE: &[u8] = &[9];
const NATIVE_TOKEN_OWNER_SUBSPACE: &[u8] = &[10];
const TRANSACTION_FILTERING_SUBSPACE: &[u8] = &[11];

/// The maximum ArbOS version supported by this node.
pub const MAX_ARBOS_VERSION_SUPPORTED: u64 = 60;

/// Central ArbOS state aggregating all subsystem states.
pub struct ArbosState<D, B: Burner> {
    pub arbos_version: u64,
    pub max_arbos_version_supported: u64,
    pub upgrade_version: StorageBackedUint64<D>,
    pub upgrade_timestamp: StorageBackedUint64<D>,
    pub network_fee_account: StorageBackedAddress<D>,
    pub l1_pricing_state: L1PricingState<D>,
    pub l2_pricing_state: L2PricingState<D>,
    pub retryable_state: RetryableState<D>,
    pub address_table: AddressTable<D>,
    pub chain_owners: AddressSet<D>,
    pub send_merkle_accumulator: MerkleAccumulator<D>,
    pub programs: Programs<D>,
    pub blockhashes: Blockhashes<D>,
    pub chain_id: StorageBackedBigUint<D>,
    pub chain_config: StorageBackedBytes<D>,
    pub genesis_block_num: StorageBackedUint64<D>,
    pub infra_fee_account: StorageBackedAddress<D>,
    pub brotli_compression_level: StorageBackedUint64<D>,
    pub backing_storage: Storage<D>,
    pub burner: B,
    pub native_token_enabled_from_time: StorageBackedUint64<D>,
    pub native_token_owners: AddressSet<D>,
    pub transaction_filtering_enabled_from_time: StorageBackedUint64<D>,
    pub transaction_filterers: AddressSet<D>,
    pub features: Features<D>,
    pub filtered_funds_recipient: StorageBackedAddress<D>,
    pub filtered_transactions: FilteredTransactionsState<D>,
}

impl<D: Database, B: Burner> ArbosState<D, B> {
    /// Open existing ArbOS state from storage.
    pub fn open(state: *mut revm::database::State<D>, burner: B) -> Result<Self, ()> {
        let backing_storage = Storage::new(state, B256::ZERO);

        let arbos_version = backing_storage.get_uint64_by_uint64(VERSION_OFFSET)?;
        if arbos_version == 0 {
            return Err(());
        }

        let chain_config_sto = backing_storage.open_sub_storage(CHAIN_CONFIG_SUBSPACE);
        let features_sto = backing_storage.open_sub_storage(FEATURES_SUBSPACE);

        Ok(Self {
            arbos_version,
            max_arbos_version_supported: MAX_ARBOS_VERSION_SUPPORTED,
            upgrade_version: StorageBackedUint64::new(state, B256::ZERO, UPGRADE_VERSION_OFFSET),
            upgrade_timestamp: StorageBackedUint64::new(
                state,
                B256::ZERO,
                UPGRADE_TIMESTAMP_OFFSET,
            ),
            network_fee_account: StorageBackedAddress::new(
                state,
                B256::ZERO,
                NETWORK_FEE_ACCOUNT_OFFSET,
            ),
            l1_pricing_state: L1PricingState::open(
                backing_storage.open_sub_storage(L1_PRICING_SUBSPACE),
                arbos_version,
            ),
            l2_pricing_state: L2PricingState::open(
                backing_storage.open_sub_storage(L2_PRICING_SUBSPACE),
                arbos_version,
            ),
            retryable_state: RetryableState::open(
                backing_storage.open_sub_storage(RETRYABLES_SUBSPACE),
            ),
            address_table: address_table::open_address_table(
                backing_storage.open_sub_storage(ADDRESS_TABLE_SUBSPACE),
            ),
            chain_owners: address_set::open_address_set(
                backing_storage.open_sub_storage(CHAIN_OWNER_SUBSPACE),
            ),
            send_merkle_accumulator: merkle_accumulator::open_merkle_accumulator(
                backing_storage.open_sub_storage(SEND_MERKLE_SUBSPACE),
            ),
            programs: Programs::open(
                arbos_version,
                backing_storage.open_sub_storage(PROGRAMS_SUBSPACE),
            ),
            blockhashes: blockhash::open_blockhashes(
                backing_storage.open_sub_storage(BLOCKHASHES_SUBSPACE),
            ),
            chain_id: StorageBackedBigUint::new(state, B256::ZERO, CHAIN_ID_OFFSET),
            chain_config: StorageBackedBytes::new(chain_config_sto),
            genesis_block_num: StorageBackedUint64::new(
                state,
                B256::ZERO,
                GENESIS_BLOCK_NUM_OFFSET,
            ),
            infra_fee_account: StorageBackedAddress::new(
                state,
                B256::ZERO,
                INFRA_FEE_ACCOUNT_OFFSET,
            ),
            brotli_compression_level: StorageBackedUint64::new(
                state,
                B256::ZERO,
                BROTLI_COMPRESSION_LEVEL_OFFSET,
            ),
            native_token_enabled_from_time: StorageBackedUint64::new(
                state,
                B256::ZERO,
                NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET,
            ),
            native_token_owners: address_set::open_address_set(
                backing_storage.open_sub_storage(NATIVE_TOKEN_OWNER_SUBSPACE),
            ),
            transaction_filtering_enabled_from_time: StorageBackedUint64::new(
                state,
                B256::ZERO,
                TRANSACTION_FILTERING_ENABLED_FROM_TIME_OFFSET,
            ),
            transaction_filterers: address_set::open_address_set(
                backing_storage.open_sub_storage(TRANSACTION_FILTERING_SUBSPACE),
            ),
            features: features::open_features(
                state,
                features_sto.base_key(),
                0,
            ),
            filtered_funds_recipient: StorageBackedAddress::new(
                state,
                B256::ZERO,
                FILTERED_FUNDS_RECIPIENT_OFFSET,
            ),
            filtered_transactions: FilteredTransactionsState::open(
                Storage::new_with_account(state, B256::ZERO, FILTERED_TX_STATE_ADDRESS),
            ),
            backing_storage,
            burner,
        })
    }

    // --- Accessor methods ---

    pub fn arbos_version(&self) -> u64 {
        self.arbos_version
    }

    pub fn backing_storage(&self) -> &Storage<D> {
        &self.backing_storage
    }

    pub fn set_format_version(&mut self, version: u64) -> Result<(), ()> {
        self.arbos_version = version;
        self.backing_storage
            .set_by_uint64(VERSION_OFFSET, B256::from(U256::from(version)))
    }

    pub fn brotli_compression_level(&self) -> Result<u64, ()> {
        self.brotli_compression_level.get()
    }

    pub fn set_brotli_compression_level(&self, level: u64) -> Result<(), ()> {
        self.brotli_compression_level.set(level)
    }

    pub fn chain_id(&self) -> Result<U256, ()> {
        self.chain_id.get()
    }

    pub fn chain_config(&self) -> Result<Vec<u8>, ()> {
        self.chain_config.get()
    }

    pub fn set_chain_config(&self, config: &[u8]) -> Result<(), ()> {
        self.chain_config.set(config)
    }

    pub fn genesis_block_num(&self) -> Result<u64, ()> {
        self.genesis_block_num.get()
    }

    pub fn network_fee_account(&self) -> Result<Address, ()> {
        self.network_fee_account.get()
    }

    pub fn set_network_fee_account(&self, account: Address) -> Result<(), ()> {
        self.network_fee_account.set(account)
    }

    pub fn infra_fee_account(&self) -> Result<Address, ()> {
        self.infra_fee_account.get()
    }

    pub fn set_infra_fee_account(&self, account: Address) -> Result<(), ()> {
        self.infra_fee_account.set(account)
    }

    pub fn filtered_funds_recipient(&self) -> Result<Address, ()> {
        self.filtered_funds_recipient.get()
    }

    pub fn filtered_funds_recipient_or_default(&self) -> Result<Address, ()> {
        let addr = self.filtered_funds_recipient.get()?;
        if addr == Address::ZERO {
            self.network_fee_account()
        } else {
            Ok(addr)
        }
    }

    pub fn set_filtered_funds_recipient(&self, addr: Address) -> Result<(), ()> {
        self.filtered_funds_recipient.set(addr)
    }

    pub fn native_token_management_from_time(&self) -> Result<u64, ()> {
        self.native_token_enabled_from_time.get()
    }

    pub fn set_native_token_management_from_time(&self, time: u64) -> Result<(), ()> {
        self.native_token_enabled_from_time.set(time)
    }

    pub fn transaction_filtering_from_time(&self) -> Result<u64, ()> {
        self.transaction_filtering_enabled_from_time.get()
    }

    pub fn set_transaction_filtering_from_time(&self, time: u64) -> Result<(), ()> {
        self.transaction_filtering_enabled_from_time.set(time)
    }

    pub fn get_scheduled_upgrade(&self) -> Result<(u64, u64), ()> {
        let version = self.upgrade_version.get()?;
        let timestamp = self.upgrade_timestamp.get()?;
        Ok((version, timestamp))
    }

    pub fn schedule_arbos_upgrade(&self, version: u64, timestamp: u64) -> Result<(), ()> {
        self.upgrade_version.set(version)?;
        self.upgrade_timestamp.set(timestamp)
    }

    /// Checks and performs a scheduled ArbOS version upgrade if due.
    pub fn upgrade_arbos_version_if_necessary(
        &mut self,
        current_timestamp: u64,
    ) -> Result<(), ()> {
        let scheduled_version = self.upgrade_version.get()?;
        let scheduled_timestamp = self.upgrade_timestamp.get()?;

        // Check: arbosVersion < upgradeTo && currentTimestamp >= flagday
        if scheduled_version == 0
            || self.arbos_version >= scheduled_version
            || current_timestamp < scheduled_timestamp
        {
            return Ok(());
        }

        if scheduled_version > MAX_ARBOS_VERSION_SUPPORTED {
            return Err(());
        }

        let old_version = self.arbos_version;
        self.upgrade_arbos_version(scheduled_version, false)?;

        // The scheduled upgrade fields are NOT cleared after upgrade.
        // They remain in storage and are simply ignored because the
        // arbos_version >= scheduled_version check prevents re-upgrade.

        if old_version != self.arbos_version {
            tracing::info!(
                old_version,
                new_version = self.arbos_version,
                "ArbOS version upgraded"
            );
        }

        Ok(())
    }

    /// Performs version upgrade steps from current version up to `upgrade_to`.
    ///
    /// `first_time` is true during genesis initialization, which affects
    /// some initial parameter values.
    pub fn upgrade_arbos_version(
        &mut self,
        upgrade_to: u64,
        first_time: bool,
    ) -> Result<(), ()> {
        while self.arbos_version < upgrade_to {
            let next = self.arbos_version + 1;

            match next {
                2 => {
                    self.l1_pricing_state.set_last_surplus(U256::ZERO, false)?;
                }
                3 => {
                    self.l1_pricing_state.set_per_batch_gas_cost(0)?;
                    self.l1_pricing_state.set_amortized_cost_cap_bips(u64::MAX)?;
                }
                4 | 5 | 6 | 7 | 8 | 9 => {
                    // No state changes needed
                }
                10 => {
                    let state = unsafe { &mut *self.backing_storage.state };
                    let pool_balance =
                        get_account_balance(state, l1_pricing::L1_PRICER_FUNDS_POOL_ADDRESS);
                    self.l1_pricing_state.set_l1_fees_available(pool_balance)?;
                }
                11 => {
                    self.l1_pricing_state
                        .set_per_batch_gas_cost(l1_pricing::INITIAL_PER_BATCH_GAS_COST_V12)?;

                    // Fix: math.MaxUint64 was incorrectly used for "disabled";
                    // the correct disable value is 0.
                    let old_cap = self.l1_pricing_state.amortized_cost_cap_bips()?;
                    if old_cap == u64::MAX {
                        self.l1_pricing_state.set_amortized_cost_cap_bips(0)?;
                    }

                    if !first_time {
                        self.chain_owners.clear_list()?;
                    }
                }
                // 12..=19: reserved for Orbit chains
                12..=19 => {}
                20 => {
                    self.set_brotli_compression_level(1)?;
                }
                // 21..=29: reserved for Orbit chains
                21..=29 => {}
                30 => {
                    Programs::initialize(
                        next,
                        &self.backing_storage.open_sub_storage(PROGRAMS_SUBSPACE),
                    );
                }
                31 => {
                    let mut params = self.programs.params()?;
                    params.upgrade_to_version(2).map_err(|_| ())?;
                    params
                        .save(&self.programs.backing_storage.open_sub_storage(&[0]))
                        .map_err(|_| ())?;
                }
                32 => {
                    // No state changes needed
                }
                // 33..=39: reserved for Orbit chains
                33..=39 => {}
                40 => {
                    // EIP-2935: historical block hashes
                    let state = unsafe { &mut *self.backing_storage.state };
                    set_account_nonce(state, HISTORY_STORAGE_ADDRESS, 1);
                    set_account_code(
                        state,
                        HISTORY_STORAGE_ADDRESS,
                        HISTORY_STORAGE_CODE_ARBITRUM.clone(),
                    );
                    // Upgrade Stylus params for version 40
                    let mut params = self.programs.params()?;
                    params.upgrade_to_arbos_version(next).map_err(|_| ())?;
                    params
                        .save(&self.programs.backing_storage.open_sub_storage(&[0]))
                        .map_err(|_| ())?;
                }
                41 => {
                    // No state changes needed
                }
                // 42..=49: reserved for Orbit chains
                42..=49 => {}
                50 => {
                    let mut params = self.programs.params()?;
                    params.upgrade_to_arbos_version(next).map_err(|_| ())?;
                    params
                        .save(&self.programs.backing_storage.open_sub_storage(&[0]))
                        .map_err(|_| ())?;
                    self.l2_pricing_state
                        .set_max_per_tx_gas_limit(l2_pricing::INITIAL_PER_TX_GAS_LIMIT_V50)?;
                }
                51 => {
                    // No state changes needed
                }
                // 52..=59: reserved for Orbit chains
                52..=59 => {}
                60 => {
                    let mut params = self.programs.params()?;
                    params.upgrade_to_arbos_version(next).map_err(|_| ())?;
                    params
                        .save(&self.programs.backing_storage.open_sub_storage(&[0]))
                        .map_err(|_| ())?;
                    // Initialize transaction filtering address set
                    crate::address_set::initialize_address_set(
                        &self
                            .backing_storage
                            .open_sub_storage(TRANSACTION_FILTERING_SUBSPACE),
                    )?;
                }
                _ => {
                    tracing::error!(version = next, "unsupported ArbOS version");
                    return Err(());
                }
            }

            // Install precompile code for newly introduced precompiles
            for &(addr, version) in PRECOMPILE_MIN_ARBOS_VERSIONS {
                if version == next {
                    let state = unsafe { &mut *self.backing_storage.state };
                    set_account_code(state, addr, Bytes::from_static(&[0xFE])); // INVALID opcode
                }
            }

            self.arbos_version = next;
            self.programs.arbos_version = next;
            self.l1_pricing_state.arbos_version = next;
            self.l2_pricing_state.arbos_version = next;
        }

        // First-time initialization overrides
        if first_time && upgrade_to >= 6 {
            if upgrade_to < 11 {
                self.l1_pricing_state
                    .set_per_batch_gas_cost(l1_pricing::INITIAL_PER_BATCH_GAS_COST_V6)?;
            }
            self.l1_pricing_state
                .set_equilibration_units(U256::from(l1_pricing::INITIAL_EQUILIBRATION_UNITS_V6))?;
            self.l2_pricing_state
                .set_speed_limit_per_second(l2_pricing::INITIAL_SPEED_LIMIT_PER_SECOND_V6)?;
            self.l2_pricing_state
                .set_max_per_block_gas_limit(l2_pricing::INITIAL_PER_BLOCK_GAS_LIMIT_V6)?;
        }

        // Persist the final version
        self.set_format_version(self.arbos_version)?;

        Ok(())
    }
}
