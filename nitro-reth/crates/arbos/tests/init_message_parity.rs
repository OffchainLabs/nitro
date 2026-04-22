use alloy_primitives::U256;
use arbos::arbos_types::{parse_init_message, DEFAULT_INITIAL_L1_BASE_FEE};

/// Nitro `ParseInitMessage` format (`arbos/arbostypes/incomingmessage.go:325`):
///   - 32 bytes: chain_id only, base_fee = DefaultInitialL1BaseFee
///   - >32 bytes: chain_id (32) + version (byte 32) + version-specific fields
///   - empty / other lengths: error
///
/// Our `parse_init_message` reads version FIRST. This is a divergence.
#[test]
fn nitro_format_32_bytes_returns_chain_id_only() {
    let chain_id_value = 421614u64;
    let mut data = [0u8; 32];
    data[24..32].copy_from_slice(&chain_id_value.to_be_bytes());

    let parsed = parse_init_message(&data).expect("parse");
    assert_eq!(
        parsed.chain_id,
        U256::from(chain_id_value),
        "Nitro: 32-byte init message contains only chain_id"
    );
    assert_eq!(
        parsed.initial_l1_base_fee,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE),
        "Nitro: base fee defaults when not present"
    );
}

/// Nitro v0: chain_id (32) + version=0 (1) + chain_config
#[test]
fn nitro_format_v0_with_chain_config() {
    let chain_id_value = 421614u64;
    let mut data = Vec::new();
    let mut chain_id_bytes = [0u8; 32];
    chain_id_bytes[24..32].copy_from_slice(&chain_id_value.to_be_bytes());
    data.extend_from_slice(&chain_id_bytes);
    data.push(0u8);
    data.extend_from_slice(b"{\"chainId\":421614}");

    let parsed = parse_init_message(&data).expect("parse");
    assert_eq!(parsed.chain_id, U256::from(chain_id_value));
    assert_eq!(parsed.serialized_chain_config, b"{\"chainId\":421614}");
}

/// Nitro v1: chain_id (32) + version=1 (1) + l1_base_fee (32) + chain_config
#[test]
fn nitro_format_v1_with_base_fee_and_config() {
    let chain_id_value = 421614u64;
    let custom_base_fee = 7u64 * 1_000_000_000u64;

    let mut data = Vec::new();
    let mut chain_id_bytes = [0u8; 32];
    chain_id_bytes[24..32].copy_from_slice(&chain_id_value.to_be_bytes());
    data.extend_from_slice(&chain_id_bytes);
    data.push(1u8);
    let mut base_fee_bytes = [0u8; 32];
    base_fee_bytes[24..32].copy_from_slice(&custom_base_fee.to_be_bytes());
    data.extend_from_slice(&base_fee_bytes);
    data.extend_from_slice(b"{\"chainId\":421614}");

    let parsed = parse_init_message(&data).expect("parse");
    assert_eq!(parsed.chain_id, U256::from(chain_id_value));
    assert_eq!(parsed.initial_l1_base_fee, U256::from(custom_base_fee));
    assert_eq!(parsed.serialized_chain_config, b"{\"chainId\":421614}");
}

/// Nitro errors on empty data.
#[test]
fn nitro_format_empty_message_is_error() {
    assert!(
        parse_init_message(&[]).is_err(),
        "Nitro errors on empty init message; ours returns defaults"
    );
}
