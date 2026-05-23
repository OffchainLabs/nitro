//! In-memory ArbOS state for unit tests.

use alloy_primitives::{Address, B256, U256};
use arb_storage::{Storage, ARBOS_STATE_ADDRESS};
use arbos::{
    arbos_state::{initialize::bootstrap, ArbosState},
    burn::SystemBurner,
    l1_pricing::L1PricingState,
    l2_pricing::L2PricingState,
    retryables::RetryableState,
};
use revm::database::{State, StateBuilder};

use crate::db::{ensure_cache_account, EmptyDb};

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

        bootstrap(
            &mut self.state,
            self.chain_id,
            self.network_fee_account,
            self.infra_fee_account,
            self.l1_initial_base_fee,
            self.arbos_version,
            SystemBurner::new(None, false),
        )
        .expect("bootstrap ArbOS state");

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
        ArbosState::open(&mut self.state, SystemBurner::new(None, false)).expect("open arbos state")
    }

    pub fn l1_pricing_state(&mut self) -> L1PricingState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        ArbosState::open(&mut self.state, SystemBurner::new(None, false))
            .expect("open arbos state")
            .l1_pricing_state
    }

    pub fn l2_pricing_state(&mut self) -> L2PricingState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        ArbosState::open(&mut self.state, SystemBurner::new(None, false))
            .expect("open arbos state")
            .l2_pricing_state
    }

    pub fn retryable_state(&mut self) -> RetryableState<EmptyDb> {
        assert!(self.initialized, "call initialize() first");
        ArbosState::open(&mut self.state, SystemBurner::new(None, false))
            .expect("open arbos state")
            .retryable_state
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
        let state_ptr = h.state_ptr();
        let l1 = h.l1_pricing_state();
        assert_eq!(l1.last_update_time(unsafe { &mut *state_ptr }).unwrap(), 0);
    }

    #[test]
    fn l1_pricing_state_starts_at_configured_base_fee() {
        let initial = U256::from(123u64) * U256::from(1_000_000_000u64);
        let mut h = ArbosHarness::new()
            .with_l1_initial_base_fee(initial)
            .initialize();
        let state_ptr = h.state_ptr();
        let l1 = h.l1_pricing_state();
        assert_eq!(
            l1.price_per_unit(unsafe { &mut *state_ptr }).unwrap(),
            initial
        );
    }

    #[test]
    fn chain_id_round_trips() {
        let mut h = ArbosHarness::new().with_chain_id(421614).initialize();
        let state_ptr = h.state_ptr();
        let s = h.arbos_state();
        assert_eq!(
            s.chain_id(unsafe { &mut *state_ptr }).unwrap(),
            U256::from(421614u64)
        );
    }
}
