use alloy_primitives::{Address, U256};
use revm::Database;

use arb_storage::{Storage, StorageBackedAddress, StorageBackedBigInt};
use crate::address_set::AddressSet;

const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const POSTER_ADDRS_KEY: &[u8] = &[0];
const POSTER_INFO_KEY: &[u8] = &[1];

const TOTAL_FUNDS_DUE_OFFSET: u64 = 0;
const FUNDS_DUE_OFFSET: u64 = 0;
const PAY_TO_OFFSET: u64 = 1;

pub struct BatchPostersTable<D> {
    poster_addrs: AddressSet<D>,
    poster_info: Storage<D>,
    pub total_funds_due: StorageBackedBigInt<D>,
}

pub struct BatchPosterState<D> {
    funds_due: StorageBackedBigInt<D>,
    pay_to: StorageBackedAddress<D>,
}

pub struct FundsDueItem {
    pub address: Address,
    pub funds_due: U256,
}

pub fn initialize_batch_posters_table<D: Database>(
    l1_pricing_storage: &Storage<D>,
    initial_poster: Address,
) {
    let bpt_storage = l1_pricing_storage.open_sub_storage(BATCH_POSTER_TABLE_KEY);
    let poster_addrs_storage = bpt_storage.open_sub_storage(POSTER_ADDRS_KEY);
    let poster_info = bpt_storage.open_sub_storage(POSTER_INFO_KEY);

    let addrs = crate::address_set::open_address_set(poster_addrs_storage);
    let _ = addrs.add(initial_poster);

    let bp_storage = poster_info.open_sub_storage(initial_poster.as_slice());
    let pay_to = StorageBackedAddress::new(bp_storage.state_ptr(), bp_storage.base_key(), PAY_TO_OFFSET);
    let _ = pay_to.set(initial_poster);

    let funds_due = StorageBackedBigInt::new(bp_storage.state_ptr(), bp_storage.base_key(), FUNDS_DUE_OFFSET);
    let _ = funds_due.set(U256::ZERO);

    let total_funds_due = StorageBackedBigInt::new(bpt_storage.state_ptr(), bpt_storage.base_key(), TOTAL_FUNDS_DUE_OFFSET);
    let _ = total_funds_due.set(U256::ZERO);
}

pub fn open_batch_posters_table<D: Database>(l1_pricing_storage: &Storage<D>) -> BatchPostersTable<D> {
    let bpt_storage = l1_pricing_storage.open_sub_storage(BATCH_POSTER_TABLE_KEY);
    let poster_addrs_storage = bpt_storage.open_sub_storage(POSTER_ADDRS_KEY);
    let poster_info = bpt_storage.open_sub_storage(POSTER_INFO_KEY);

    let poster_addrs = crate::address_set::open_address_set(poster_addrs_storage);
    let total_funds_due = StorageBackedBigInt::new(
        bpt_storage.state_ptr(),
        bpt_storage.base_key(),
        TOTAL_FUNDS_DUE_OFFSET,
    );

    BatchPostersTable {
        poster_addrs,
        poster_info,
        total_funds_due,
    }
}

impl<D: Database> BatchPostersTable<D> {
    pub fn open(l1_pricing_storage: &Storage<D>) -> Self {
        open_batch_posters_table(l1_pricing_storage)
    }

    pub fn contains_poster(&self, poster: Address) -> Result<bool, ()> {
        self.poster_addrs.is_member(poster)
    }

    pub fn open_poster(
        &self,
        poster: Address,
        create_if_not_exist: bool,
    ) -> Result<BatchPosterState<D>, ()> {
        let is_poster = self.poster_addrs.is_member(poster)?;
        if !is_poster {
            if !create_if_not_exist {
                return Err(());
            }
            return self.add_poster(poster, poster);
        }
        Ok(self.internal_open(poster))
    }

    pub fn add_poster(
        &self,
        poster_address: Address,
        pay_to: Address,
    ) -> Result<BatchPosterState<D>, ()> {
        let is_poster = self.poster_addrs.is_member(poster_address)?;
        if is_poster {
            return Err(());
        }

        let bp_state = self.internal_open(poster_address);
        bp_state.funds_due.set(U256::ZERO)?;
        bp_state.pay_to.set(pay_to)?;
        self.poster_addrs.add(poster_address)?;
        Ok(bp_state)
    }

    fn internal_open(&self, poster: Address) -> BatchPosterState<D> {
        let bp_storage = self.poster_info.open_sub_storage(poster.as_slice());
        BatchPosterState {
            funds_due: StorageBackedBigInt::new(
                bp_storage.state_ptr(),
                bp_storage.base_key(),
                FUNDS_DUE_OFFSET,
            ),
            pay_to: StorageBackedAddress::new(
                bp_storage.state_ptr(),
                bp_storage.base_key(),
                PAY_TO_OFFSET,
            ),
        }
    }

    pub fn all_posters(&self) -> Result<Vec<Address>, ()> {
        self.poster_addrs.all_members(u64::MAX)
    }

    pub fn total_funds_due(&self) -> Result<U256, ()> {
        self.total_funds_due.get_raw()
    }

    pub fn get_funds_due_list(&self) -> Result<Vec<FundsDueItem>, ()> {
        let posters = self.all_posters()?;
        let mut result = Vec::new();
        for poster in posters {
            let state = self.internal_open(poster);
            let due = state.funds_due()?;
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

impl<D: Database> BatchPosterState<D> {
    pub fn funds_due(&self) -> Result<U256, ()> {
        self.funds_due.get_raw()
    }

    pub fn set_funds_due(
        &self,
        value: U256,
        total_funds_due: &StorageBackedBigInt<D>,
    ) -> Result<(), ()> {
        let prev = self.funds_due.get_raw().unwrap_or(U256::ZERO);
        let prev_total = total_funds_due.get_raw().unwrap_or(U256::ZERO);
        let new_total = prev_total.saturating_add(value).saturating_sub(prev);
        total_funds_due.set(new_total)?;
        self.funds_due.set(value)
    }

    pub fn pay_to(&self) -> Result<Address, ()> {
        self.pay_to.get()
    }

    pub fn set_pay_to(&self, addr: Address) -> Result<(), ()> {
        self.pay_to.set(addr)
    }
}
