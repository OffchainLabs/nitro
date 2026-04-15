//! Constants for eip-7928. Chosen to support a 630 million gas limit.

use alloy_primitives::{B256, b256};

/// Maximum number of transactions per block.
pub const MAX_TXS_PER_BLOCK: usize = 30_000;

/// Maximum number of unique storage slots modified in a block.
pub const MAX_SLOTS: usize = 300_000;

/// Maximum number of unique accounts accessed in a block.
pub const MAX_ACCOUNTS: usize = 300_000;

/// Maximum contract bytecode size in bytes.
pub const MAX_CODE_SIZE: usize = 24_576;

/// Type alias for block index for eip-7928.
pub type BlockAccessIndex = u64;

/// The empty block access list hash.
pub const EMPTY_BLOCK_ACCESS_LIST_HASH: B256 =
    b256!("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347");
