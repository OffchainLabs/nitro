use alloy_primitives::{Address, U256};

/// Transfers balance between two addresses.
///
/// If `from` is None, this is a mint operation.
/// If `to` is None, this is a burn operation.
pub fn transfer_balance<F>(
    from: Option<&Address>,
    to: Option<&Address>,
    amount: U256,
    mut state_fn: F,
) -> Result<(), ()>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), ()>,
{
    state_fn(from, to, amount)
}

/// Mints balance to the given address.
pub fn mint_balance<F>(to: &Address, amount: U256, state_fn: F) -> Result<(), ()>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), ()>,
{
    transfer_balance(None, Some(to), amount, state_fn)
}

/// Burns balance from the given address.
pub fn burn_balance<F>(from: &Address, amount: U256, state_fn: F) -> Result<(), ()>
where
    F: FnMut(Option<&Address>, Option<&Address>, U256) -> Result<(), ()>,
{
    transfer_balance(Some(from), None, amount, state_fn)
}
