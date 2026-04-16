//! In-memory ArbOS state for unit tests.

use alloy_primitives::{Address, B256, U256};
use arb_storage::{
    set_account_nonce, Storage, StorageBackedAddress, ARBOS_STATE_ADDRESS,
};
use arbos::{
    arbos_state::ArbosState,
    burn::SystemBurner,
    l1_pricing::{self, L1PricingState},
    l2_pricing::{self, L2PricingState},
    retryables::{self, RetryableState},
};
use revm::database::{State, StateBuilder};

use crate::db::{ensure_cache_account, EmptyDb};

const VERSION_OFFSET: u64 = 0;
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const CHAIN_ID_OFFSET: u64 = 4;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;

const L1_PRICING_SUBSPACE: &[u8] = &[0];
const L2_PRICING_SUBSPACE: &[u8] = &[1];
const RETRYABLES_SUBSPACE: &[u8] = &[2];

/// Builder + handle for an in-memory ArbOS state.
pub struct ArbosHarness {
    state: Box<State<EmptyDb>>,
    arbos_version: u64,
    chain_id: u64,
    network_fee_account: Address,
    infra_fee_account: Address,
    l1_initial_base_fee: U256,
    initialized: bool,
}

impl Default for ArbosHarness {
    fn default() -> Self {
        Self::new()
    }
}

impl ArbosHarness {
    /// Defaults: ArbOS v30, chain id 412346, L1 base fee 0.1 gwei.
    pub fn new() -> Self {
        let state = Box::new(
            StateBuilder::new()
                .with_database(EmptyDb)
                .with_bundle_update()
                .build(),
        );
        Self {
            state,
            arbos_version: 30,
            chain_id: 412346,
            network_fee_account: Address::ZERO,
            infra_fee_account: Address::ZERO,
            l1_initial_base_fee: U256::from(100_000_000u64),
            initialized: false,
        }
    }

    pub fn with_arbos_version(mut self, v: u64) -> Self {
        assert!(!self.initialized, "set version before initialize()");
        self.arbos_version = v;
        self
    }

    pub fn with_chain_id(mut self, id: u64) -> Self {
        assert!(!self.initialized, "set chain id before initialize()");
        self.chain_id = id;
        self
    }

    pub fn with_network_fee_account(mut self, a: Address) -> Self {
        assert!(!self.initialized, "set fee account before initialize()");
        self.network_fee_account = a;
        self
    }

    pub fn with_infra_fee_account(mut self, a: Address) -> Self {
        assert!(!self.initialized, "set fee account before initialize()");
        self.infra_fee_account = a;
        self
    }

    pub fn with_l1_initial_base_fee(mut self, fee: U256) -> Self {
        assert!(!self.initialized, "set base fee before initialize()");
        self.l1_initial_base_fee = fee;
        self
    }

    pub fn initialize(mut self) -> Self {
        assert!(!self.initialized, "initialize() called twice");

        ensure_cache_account(&mut self.state, ARBOS_STATE_ADDRESS);
        set_account_nonce(&mut self.state, ARBOS_STATE_ADDRESS, 1);

        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();

        let backing = Storage::new(state_ptr, B256::ZERO);
        backing
            .set_by_uint64(VERSION_OFFSET, B256::from(U256::from(1u64)))
            .expect("set initial version");
        backing
            .set_by_uint64(CHAIN_ID_OFFSET, B256::from(U256::from(self.chain_id)))
            .expect("set chain id");
        StorageBackedAddress::new(state_ptr, B256::ZERO, NETWORK_FEE_ACCOUNT_OFFSET)
            .set(self.network_fee_account)
            .expect("set network fee account");
        StorageBackedAddress::new(state_ptr, B256::ZERO, INFRA_FEE_ACCOUNT_OFFSET)
            .set(self.infra_fee_account)
            .expect("set infra fee account");

        l1_pricing::initialize_l1_pricing_state(
            &backing.open_sub_storage(L1_PRICING_SUBSPACE),
            self.network_fee_account,
            self.l1_initial_base_fee,
        );
        l2_pricing::initialize_l2_pricing_state(&backing.open_sub_storage(L2_PRICING_SUBSPACE));
        retryables::initialize_retryable_state(&backing.open_sub_storage(RETRYABLES_SUBSPACE))
            .expect("init retryables");

        let mut state = ArbosState::<EmptyDb, SystemBurner>::open(
            state_ptr,
            SystemBurner::new(None, false),
        )
        .expect("open arbos state at v1");
        state
            .upgrade_arbos_version(self.arbos_version, true)
            .expect("upgrade to target arbos version");

        self.initialized = true;
        self
    }

    pub fn state(&mut self) -> &mut State<EmptyDb> {
        &mut self.state
    }

    pub fn state_ptr(&mut self) -> *mut State<EmptyDb> {
        self.state.as_mut()
    }

    pub fn arbos_state(&mut self) -> ArbosState<EmptyDb, SystemBurner> {
        assert!(self.initialized, "call initialize() first");
        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();
        ArbosState::open(state_ptr, SystemBurner::new(None, false))
            .expect("open arbos state")
    }

    pub fn l1_pricing_state(&mut self) -> L1PricingState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();
        let backing = Storage::new(state_ptr, B256::ZERO);
        L1PricingState::open(
            backing.open_sub_storage(L1_PRICING_SUBSPACE),
            self.arbos_version,
        )
    }

    pub fn l2_pricing_state(&mut self) -> L2PricingState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();
        let backing = Storage::new(state_ptr, B256::ZERO);
        L2PricingState::open(
            backing.open_sub_storage(L2_PRICING_SUBSPACE),
            self.arbos_version,
        )
    }

    pub fn retryable_state(&mut self) -> RetryableState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();
        let backing = Storage::new(state_ptr, B256::ZERO);
        RetryableState::open(backing.open_sub_storage(RETRYABLES_SUBSPACE))
    }

    pub fn root_storage(&mut self) -> Storage<EmptyDb> {
        let state_ptr: *mut State<EmptyDb> = self.state.as_mut();
        Storage::new(state_ptr, B256::ZERO)
    }

    pub fn arbos_version(&self) -> u64 {
        self.arbos_version
    }

    pub fn chain_id(&self) -> u64 {
        self.chain_id
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn harness_initializes_at_v30() {
        let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
        let s = h.arbos_state();
        assert_eq!(s.arbos_version(), 30);
    }

    #[test]
    fn harness_initializes_at_v60() {
        let mut h = ArbosHarness::new().with_arbos_version(60).initialize();
        let s = h.arbos_state();
        assert_eq!(s.arbos_version(), 60);
    }

    #[test]
    fn l1_pricing_state_starts_at_zero_last_update_time() {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        assert_eq!(l1.last_update_time().unwrap(), 0);
    }

    #[test]
    fn l1_pricing_state_starts_at_configured_base_fee() {
        let initial = U256::from(123u64) * U256::from(1_000_000_000u64);
        let mut h = ArbosHarness::new()
            .with_l1_initial_base_fee(initial)
            .initialize();
        let l1 = h.l1_pricing_state();
        assert_eq!(l1.price_per_unit().unwrap(), initial);
    }

    #[test]
    fn chain_id_round_trips() {
        let mut h = ArbosHarness::new().with_chain_id(421614).initialize();
        let s = h.arbos_state();
        assert_eq!(s.chain_id().unwrap(), U256::from(421614u64));
    }
}
