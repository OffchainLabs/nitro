use alloy_primitives::{Address, U256};
use revm::Database;

use crate::address_set::AddressSet;
use arb_storage::{Storage, StorageBackedAddress, StorageBackedBigInt, StorageBackend};

use super::L1PricingError;

const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const POSTER_ADDRS_KEY: &[u8] = &[0];
const POSTER_INFO_KEY: &[u8] = &[1];

const TOTAL_FUNDS_DUE_OFFSET: u64 = 0;
const FUNDS_DUE_OFFSET: u64 = 0;
const PAY_TO_OFFSET: u64 = 1;

pub struct BatchPostersTable<'a, D> {
    poster_addrs: AddressSet<'a, D>,
    poster_info: Storage<'a, D>,
    pub total_funds_due: StorageBackedBigInt,
}

pub struct BatchPosterState {
    funds_due: StorageBackedBigInt,
    pay_to: StorageBackedAddress,
}

pub struct FundsDueItem {
    pub address: Address,
    pub funds_due: U256,
}

pub fn initialize_batch_posters_table<D: Database, B: StorageBackend>(
    l1_pricing_storage: &Storage<'_, D>,
    backend: &mut B,
    initial_poster: Address,
) -> Result<(), L1PricingError> {
    let bpt_storage = l1_pricing_storage.open_sub_storage(BATCH_POSTER_TABLE_KEY);
    let poster_addrs_storage = bpt_storage.open_sub_storage(POSTER_ADDRS_KEY);
    let poster_info = bpt_storage.open_sub_storage(POSTER_INFO_KEY);

    let addrs = crate::address_set::open_address_set(poster_addrs_storage);
    addrs.add(backend, initial_poster)?;

    let bp_storage = poster_info.open_sub_storage(initial_poster.as_slice());
    let pay_to = StorageBackedAddress::new(bp_storage.base_key(), PAY_TO_OFFSET);
    pay_to.set(backend, initial_poster)?;

    let funds_due = StorageBackedBigInt::new(bp_storage.base_key(), FUNDS_DUE_OFFSET);
    funds_due.set(backend, U256::ZERO)?;

    let total_funds_due = StorageBackedBigInt::new(bpt_storage.base_key(), TOTAL_FUNDS_DUE_OFFSET);
    total_funds_due.set(backend, U256::ZERO)?;
    Ok(())
}

pub fn open_batch_posters_table<'a, D>(
    l1_pricing_storage: &Storage<'a, D>,
) -> BatchPostersTable<'a, D> {
    let bpt_storage = l1_pricing_storage.open_sub_storage(BATCH_POSTER_TABLE_KEY);
    let poster_addrs_storage = bpt_storage.open_sub_storage(POSTER_ADDRS_KEY);
    let poster_info = bpt_storage.open_sub_storage(POSTER_INFO_KEY);

    let poster_addrs = crate::address_set::open_address_set(poster_addrs_storage);
    let total_funds_due = StorageBackedBigInt::new(bpt_storage.base_key(), TOTAL_FUNDS_DUE_OFFSET);

    BatchPostersTable {
        poster_addrs,
        poster_info,
        total_funds_due,
    }
}

impl<'a, D> BatchPostersTable<'a, D> {
    pub fn open(l1_pricing_storage: &Storage<'a, D>) -> Self {
        open_batch_posters_table(l1_pricing_storage)
    }

    pub fn total_funds_due<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L1PricingError> {
        Ok(self.total_funds_due.get_raw(backend)?)
    }

    fn internal_open(&self, poster: Address) -> BatchPosterState {
        let bp_storage = self.poster_info.open_sub_storage(poster.as_slice());
        BatchPosterState {
            funds_due: StorageBackedBigInt::new(bp_storage.base_key(), FUNDS_DUE_OFFSET),
            pay_to: StorageBackedAddress::new(bp_storage.base_key(), PAY_TO_OFFSET),
        }
    }
}

impl<D> BatchPostersTable<'_, D> {
    pub fn contains_poster<B: StorageBackend>(
        &self,
        backend: &mut B,
        poster: Address,
    ) -> Result<bool, L1PricingError> {
        Ok(self.poster_addrs.is_member(backend, poster)?)
    }

    pub fn open_poster<B: StorageBackend>(
        &self,
        backend: &mut B,
        poster: Address,
        create_if_not_exist: bool,
    ) -> Result<BatchPosterState, L1PricingError> {
        let is_poster = self.poster_addrs.is_member(backend, poster)?;
        if !is_poster {
            if !create_if_not_exist {
                return Err(L1PricingError::BatchPosterNotFound);
            }
            return self.add_poster(backend, poster, poster);
        }
        Ok(self.internal_open(poster))
    }

    pub fn add_poster<B: StorageBackend>(
        &self,
        backend: &mut B,
        poster_address: Address,
        pay_to: Address,
    ) -> Result<BatchPosterState, L1PricingError> {
        let is_poster = self.poster_addrs.is_member(backend, poster_address)?;
        if is_poster {
            return Err(L1PricingError::BatchPosterAlreadyExists);
        }

        let bp_state = self.internal_open(poster_address);
        bp_state.funds_due.set(backend, U256::ZERO)?;
        bp_state.pay_to.set(backend, pay_to)?;
        self.poster_addrs.add(backend, poster_address)?;
        Ok(bp_state)
    }

    pub fn all_posters<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Vec<Address>, L1PricingError> {
        Ok(self.poster_addrs.all_members(backend, u64::MAX)?)
    }

    pub fn all_posters_capped<B: StorageBackend>(
        &self,
        backend: &mut B,
        max: u64,
    ) -> Result<Vec<Address>, L1PricingError> {
        Ok(self.poster_addrs.all_members(backend, max)?)
    }

    pub fn get_funds_due_list<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Vec<FundsDueItem>, L1PricingError> {
        let posters = self.all_posters(backend)?;
        let mut result = Vec::new();
        for poster in posters {
            let state = self.internal_open(poster);
            let due = state.funds_due(backend)?;
            if due > U256::ZERO {
                result.push(FundsDueItem {
                    address: poster,
                    funds_due: due,
                });
            }
        }
        Ok(result)
    }
}

impl BatchPosterState {
    pub fn funds_due<B: StorageBackend>(&self, backend: &mut B) -> Result<U256, L1PricingError> {
        Ok(self.funds_due.get_raw(backend)?)
    }

    pub fn set_funds_due<B: StorageBackend>(
        &self,
        backend: &mut B,
        value: U256,
        total_funds_due: &StorageBackedBigInt,
    ) -> Result<(), L1PricingError> {
        let prev = self.funds_due.get_raw(backend).unwrap_or(U256::ZERO);
        let prev_total = total_funds_due.get_raw(backend).unwrap_or(U256::ZERO);
        let new_total = prev_total.saturating_add(value).saturating_sub(prev);
        total_funds_due.set(backend, new_total)?;
        Ok(self.funds_due.set(backend, value)?)
    }

    pub fn pay_to<B: StorageBackend>(&self, backend: &mut B) -> Result<Address, L1PricingError> {
        Ok(self.pay_to.get(backend)?)
    }

    pub fn set_pay_to<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(), L1PricingError> {
        Ok(self.pay_to.set(backend, addr)?)
    }
}
