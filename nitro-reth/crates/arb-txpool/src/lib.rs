//! Arbitrum transaction pool.
//!
//! Provides the pooled transaction type and validator needed by the
//! node's transaction pool. Arbitrum L2 does not use blob transactions.

mod transaction;
pub use transaction::ArbPooledTransaction;
