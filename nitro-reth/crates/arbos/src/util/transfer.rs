use alloy_primitives::{Address, U256};
use arb_storage::DatabaseError;

/// Failure modes returned by the balance-mutation callback that `arbos`
/// hands state changes to.
#[derive(Debug, Clone, thiserror::Error)]
pub enum BalanceError {
    /// `from` did not hold enough funds to cover the requested movement.
    #[error("insufficient balance on {account}: available {available}, requested {requested}")]
    InsufficientBalance {
        account: Address,
        available: U256,
        requested: U256,
    },

    /// The underlying state database failed while servicing the transfer.
    #[error(transparent)]
    Database(#[from] DatabaseError),
}

/// Transfers balance between two addresses.
///
/// `from == None` is a mint, `to == None` is a burn.
pub fn transfer_balance<F>(
    from: Option<&Address>,
    to: Option<&Address>,
    amount: U256,
    mut state_fn: F,
) -> Result<(), BalanceError>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), BalanceError>,
{
    state_fn(from, to, amount)
}

/// Mints `amount` to `to`.
pub fn mint_balance<F>(to: &Address, amount: U256, state_fn: F) -> Result<(), BalanceError>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), BalanceError>,
{
    transfer_balance(None, Some(to), amount, state_fn)
}

/// Burns `amount` from `from`.
pub fn burn_balance<F>(from: &Address, amount: U256, state_fn: F) -> Result<(), BalanceError>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), BalanceError>,
{
    transfer_balance(Some(from), None, amount, state_fn)
}
