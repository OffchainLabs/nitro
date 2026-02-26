use alloy_primitives::{Address, B256, U256};

/// Well-known method selector for InternalTxStartBlock.
pub const INTERNAL_TX_START_BLOCK_METHOD_ID: [u8; 4] = [0xa0, 0x4a, 0x0f, 0x80];

/// Well-known method selector for batch posting report v1.
pub const INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID: [u8; 4] = [0xb1, 0xe1, 0x24, 0x27];

/// Well-known method selector for batch posting report v2.
pub const INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID: [u8; 4] = [0x14, 0x4e, 0x51, 0x4e];

/// Well-known addresses for Arbitrum system contracts.
pub const ARB_RETRYABLE_TX_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[18] = 0x00;
    bytes[19] = 0x6e;
    Address::new(bytes)
};

pub const ARB_SYS_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[19] = 0x64;
    Address::new(bytes)
};

/// Number of additional floor-gas tokens per gas token.
/// Used in EIP-7825 floor gas pricing.
pub fn floor_gas_additional_tokens(arbos_version: u64) -> u64 {
    if arbos_version >= 40 {
        3 // 4 total floor gas per token (1 base + 3 additional)
    } else {
        0
    }
}

/// L1 block info passed to internal transactions.
#[derive(Debug, Clone)]
pub struct L1Info {
    pub poster: Address,
    pub l1_block_number: u64,
    pub l1_timestamp: u64,
}

impl L1Info {
    pub fn new(poster: Address, l1_block_number: u64, l1_timestamp: u64) -> Self {
        Self {
            poster,
            l1_block_number,
            l1_timestamp,
        }
    }
}

/// Creates the ABI-encoded data for an InternalTxStartBlock call.
pub fn internal_tx_start_block_data(
    l1_block_number: u64,
    l1_timestamp: u64,
    l1_base_fee: U256,
) -> Vec<u8> {
    let mut data = Vec::with_capacity(4 + 32 * 3);
    data.extend_from_slice(&INTERNAL_TX_START_BLOCK_METHOD_ID);
    data.extend_from_slice(&B256::left_padding_from(&l1_block_number.to_be_bytes()).0);
    data.extend_from_slice(&B256::left_padding_from(&l1_timestamp.to_be_bytes()).0);
    data.extend_from_slice(&l1_base_fee.to_be_bytes::<32>());
    data
}

/// Event IDs for L2→L1 messages.
pub const L2_TO_L1_TRANSACTION_EVENT_ID: B256 = {
    // keccak256("L2ToL1Transaction(address,address,uint256,uint256,uint256,uint256,uint256,uint256,uint256,bytes)")
    // Pre-computed hash:
    let bytes: [u8; 32] = [
        0x5b, 0xaa, 0xbe, 0x19, 0x5c, 0x3e, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};

pub const L2_TO_L1_TX_EVENT_ID: B256 = {
    let bytes: [u8; 32] = [
        0x3e, 0x7a, 0xdf, 0x9f, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};

pub const REDEEM_SCHEDULED_EVENT_ID: B256 = {
    let bytes: [u8; 32] = [
        0x5a, 0x4c, 0x71, 0x5f, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};
