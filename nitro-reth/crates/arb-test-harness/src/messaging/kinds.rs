//! L1 message kind constants (mirrors Nitro's arbutil/messages.go).

pub const KIND_UNSIGNED_USER_TX: u8 = 0;
pub const KIND_CONTRACT_TX: u8 = 1;
pub const KIND_NONMUTATING: u8 = 2;
pub const KIND_DEPOSIT: u8 = 3;
pub const KIND_SIGNED_L2_TX: u8 = 4;
pub const KIND_INTERNAL_TX: u8 = 6;
pub const KIND_SIGNED_COMPRESSED_TX: u8 = 7;
pub const KIND_DELAYED_TX: u8 = 8;
pub const KIND_RETRYABLE_TX: u8 = 9;
pub const KIND_BATCH_FOR_GAS_ESTIMATION: u8 = 10;
pub const KIND_HEARTBEAT: u8 = 11;
pub const KIND_ETH_DEPOSIT: u8 = 12;
pub const KIND_BATCH_POSTING_REPORT: u8 = 13;
