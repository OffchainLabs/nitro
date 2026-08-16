//! Per-block ledger of expected balance changes, mirroring the `expectedBalanceDelta` accounting
//! of Go's `blockBuildState` (`arbos/block_processor.go`).

use std::cmp::Ordering;

use alloy_primitives::{U256, U512, aliases::I512};

/// Mismatch between the balance delta observed in state and the delta implied by the block's
/// deposits and withdrawals.
#[derive(thiserror::Error, Debug, Clone, PartialEq, Eq)]
#[error("unexpected balance delta {actual} (expected {expected})")]
pub struct BalanceDeltaError {
    /// Delta observed in state.
    pub actual: I512,
    /// Delta implied by tracked deposits/withdrawals.
    pub expected: I512,
}

/// Ledger of expected balance changes: L1 deposits add, L2→L1 withdrawals subtract. At block end
/// the state's actual delta must match.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct BalanceLedger {
    expected_delta: I512,
}

/// Widens a [`U256`] value losslessly; the result is non-negative and cannot overflow [`I512`].
fn to_i512(value: U256) -> I512 {
    I512::from_raw(value.to::<U512>())
}

impl BalanceLedger {
    pub fn new() -> Self {
        Self::default()
    }

    /// The delta implied by tracked deposits and withdrawals.
    pub fn expected_delta(&self) -> I512 {
        self.expected_delta
    }

    /// Tracks an L1 deposit (mints funds on L2).
    pub fn track_deposit(&mut self, value: U256) {
        self.expected_delta = self.expected_delta.saturating_add(to_i512(value));
    }

    /// Tracks an L2→L1 withdrawal (burns funds on L2).
    pub fn track_withdrawal(&mut self, value: U256) {
        self.expected_delta = self.expected_delta.saturating_sub(to_i512(value));
    }

    /// Verifies the post-block balance delta: minted funds (`actual > expected`) are a hard error,
    /// as is any mismatch in debug mode; burnt funds are only logged.
    #[allow(clippy::result_large_err)]
    pub fn verify(&self, actual: I512, debug_mode: bool) -> Result<(), BalanceDeltaError> {
        match (actual.cmp(&self.expected_delta), debug_mode) {
            (Ordering::Equal, _) => Ok(()),
            (Ordering::Greater, _) | (_, true) => Err(BalanceDeltaError {
                actual,
                expected: self.expected_delta,
            }),
            _ => {
                tracing::error!(
                    %actual,
                    expected = %self.expected_delta,
                    "unexpected balance delta (funds burnt)"
                );
                Ok(())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tracks_deposits_and_withdrawals() {
        let mut ledger = BalanceLedger::new();
        ledger.track_deposit(U256::from(1_000));
        ledger.track_deposit(U256::from(500));
        ledger.track_withdrawal(U256::from(300));
        assert_eq!(ledger.expected_delta(), I512::try_from(1_200).unwrap());
    }

    #[test]
    fn verify_accepts_exact_match() {
        let mut ledger = BalanceLedger::new();
        ledger.track_deposit(U256::from(42));
        let actual = I512::try_from(42).unwrap();
        assert!(ledger.verify(actual, false).is_ok());
        assert!(ledger.verify(actual, true).is_ok());
    }

    #[test]
    fn verify_rejects_minted_funds() {
        let ledger = BalanceLedger::new();
        let err = ledger.verify(I512::ONE, false).unwrap_err();
        assert_eq!(err.actual, I512::ONE);
        assert_eq!(err.expected, I512::ZERO);
    }

    #[test]
    fn verify_tolerates_burnt_funds_except_in_debug_mode() {
        let mut ledger = BalanceLedger::new();
        ledger.track_deposit(U256::from(100));
        let actual = I512::try_from(50).unwrap();
        // Burn (actual below expected): log-only.
        assert!(ledger.verify(actual, false).is_ok());
        // Same delta is an error in debug mode.
        assert!(ledger.verify(actual, true).is_err());
    }

    #[test]
    fn full_u256_range_is_tracked_exactly() {
        let max = to_i512(U256::MAX);

        let mut ledger = BalanceLedger::new();
        ledger.track_deposit(U256::MAX);
        assert_eq!(ledger.expected_delta(), max);
        ledger.track_deposit(U256::from(1u64));
        assert_eq!(ledger.expected_delta(), max + I512::ONE);

        let mut ledger = BalanceLedger::new();
        ledger.track_withdrawal(U256::MAX);
        ledger.track_withdrawal(U256::MAX);
        assert_eq!(ledger.expected_delta(), -(max + max));
    }
}
