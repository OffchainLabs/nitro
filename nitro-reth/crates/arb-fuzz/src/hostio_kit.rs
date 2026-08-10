use alloy_primitives::Address;

pub const SENTINEL_ADDR: Address = Address::new([
    0xde, 0xad, 0xbe, 0xef, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01,
]);

pub const SENTINEL_SLOT_U64: u64 = 0xfeed_face_dead_beef;
pub const SENTINEL_VALUE_BYTE: u8 = 0xa1;
