//! Contains the [`BlockAccessList`] type, which represents a simple list of account changes.

use crate::account_changes::AccountChanges;
use alloc::vec::Vec;

/// This struct is used to store `account_changes` in a block.
pub type BlockAccessList = Vec<AccountChanges>;

/// Computes the hash of the given block access list.
#[cfg(feature = "rlp")]
pub fn compute_block_access_list_hash(bal: &[AccountChanges]) -> alloy_primitives::B256 {
    let mut buf = Vec::new();
    alloy_rlp::encode_list(bal, &mut buf);
    alloy_primitives::keccak256(&buf)
}

/// Block-Level Access List wrapper type with helper methods for metrics and validation.
pub mod bal {
    use crate::account_changes::AccountChanges;
    use alloc::vec::{IntoIter, Vec};
    use core::{
        ops::{Deref, Index},
        slice::Iter,
    };

    /// A wrapper around [`Vec<AccountChanges>`] that provides helper methods for
    /// computing metrics and statistics about the block access list.
    ///
    /// This type implements `Deref` to `[AccountChanges]` for easy access to the
    /// underlying data while providing additional utility methods for BAL analysis.
    #[derive(Clone, Debug, Default, PartialEq, Eq)]
    #[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
    #[cfg_attr(feature = "rlp", derive(alloy_rlp::RlpEncodable, alloy_rlp::RlpDecodable))]
    pub struct Bal(Vec<AccountChanges>);

    impl From<Bal> for Vec<AccountChanges> {
        fn from(this: Bal) -> Self {
            this.0
        }
    }

    impl From<Vec<AccountChanges>> for Bal {
        fn from(list: Vec<AccountChanges>) -> Self {
            Self(list)
        }
    }

    impl Deref for Bal {
        type Target = [AccountChanges];

        fn deref(&self) -> &Self::Target {
            self.as_slice()
        }
    }

    impl IntoIterator for Bal {
        type Item = AccountChanges;
        type IntoIter = IntoIter<AccountChanges>;

        fn into_iter(self) -> Self::IntoIter {
            self.0.into_iter()
        }
    }

    impl<'a> IntoIterator for &'a Bal {
        type Item = &'a AccountChanges;
        type IntoIter = Iter<'a, AccountChanges>;

        fn into_iter(self) -> Self::IntoIter {
            self.iter()
        }
    }

    impl FromIterator<AccountChanges> for Bal {
        fn from_iter<I: IntoIterator<Item = AccountChanges>>(iter: I) -> Self {
            Self(iter.into_iter().collect())
        }
    }

    impl<I> Index<I> for Bal
    where
        I: core::slice::SliceIndex<[AccountChanges]>,
    {
        type Output = I::Output;

        #[inline]
        fn index(&self, index: I) -> &Self::Output {
            &self.0[index]
        }
    }

    impl Bal {
        /// Creates a new [`Bal`] from the provided account changes.
        pub const fn new(account_changes: Vec<AccountChanges>) -> Self {
            Self(account_changes)
        }

        /// Adds a new [`AccountChanges`] entry to the list.
        pub fn push(&mut self, account_changes: AccountChanges) {
            self.0.push(account_changes)
        }

        /// Returns `true` if the list contains no elements.
        #[inline]
        pub const fn is_empty(&self) -> bool {
            self.0.is_empty()
        }

        /// Returns the number of account change entries contained in the list.
        #[inline]
        pub const fn len(&self) -> usize {
            self.0.len()
        }

        /// Returns an iterator over the [`AccountChanges`] entries.
        #[inline]
        pub fn iter(&self) -> Iter<'_, AccountChanges> {
            self.0.iter()
        }

        /// Returns a slice of the contained [`AccountChanges`].
        #[inline]
        pub const fn as_slice(&self) -> &[AccountChanges] {
            self.0.as_slice()
        }

        /// Returns a vector of [`AccountChanges`].
        pub fn into_inner(self) -> Vec<AccountChanges> {
            self.0
        }

        /// Returns the total number of accounts with changes in this BAL.
        #[inline]
        pub const fn account_count(&self) -> usize {
            self.0.len()
        }

        /// Returns the total number of storage changes across all accounts.
        pub fn total_storage_changes(&self) -> usize {
            self.0.iter().map(|a| a.storage_changes.len()).sum()
        }

        /// Returns the total number of storage reads across all accounts.
        pub fn total_storage_reads(&self) -> usize {
            self.0.iter().map(|a| a.storage_reads.len()).sum()
        }

        /// Returns the total number of storage slots (both changes and reads) across all accounts.
        pub fn total_slots(&self) -> usize {
            self.0.iter().map(|a| a.storage_changes.len() + a.storage_reads.len()).sum()
        }

        /// Returns the total number of balance changes across all accounts.
        pub fn total_balance_changes(&self) -> usize {
            self.0.iter().map(|a| a.balance_changes.len()).sum()
        }

        /// Returns the total number of nonce changes across all accounts.
        pub fn total_nonce_changes(&self) -> usize {
            self.0.iter().map(|a| a.nonce_changes.len()).sum()
        }

        /// Returns the total number of code changes across all accounts.
        pub fn total_code_changes(&self) -> usize {
            self.0.iter().map(|a| a.code_changes.len()).sum()
        }

        /// Returns a summary of all change counts for metrics reporting.
        ///
        /// Returns a tuple of (account_changes, storage_changes, balance_changes, nonce_changes,
        /// code_changes).
        pub fn change_counts(&self) -> BalChangeCounts {
            let mut counts = BalChangeCounts::default();
            for account in &self.0 {
                counts.accounts += 1;
                counts.storage += account.storage_changes.len();
                counts.balance += account.balance_changes.len();
                counts.nonce += account.nonce_changes.len();
                counts.code += account.code_changes.len();
            }
            counts
        }

        /// Computes the hash of this block access list.
        #[cfg(feature = "rlp")]
        pub fn compute_hash(&self) -> alloy_primitives::B256 {
            super::compute_block_access_list_hash(&self.0)
        }
    }

    /// Summary of change counts in a BAL for metrics reporting.
    #[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
    pub struct BalChangeCounts {
        /// Number of accounts with changes.
        pub accounts: usize,
        /// Total number of storage changes.
        pub storage: usize,
        /// Total number of balance changes.
        pub balance: usize,
        /// Total number of nonce changes.
        pub nonce: usize,
        /// Total number of code changes.
        pub code: usize,
    }
}
