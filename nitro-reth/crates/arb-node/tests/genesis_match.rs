//! Verifies that arbreth's genesis block matches the header that
//! Nitro's `MakeGenesisBlock` would produce for the same chain config.
//!
//! Nitro hardcodes the genesis header in `arbosState.MakeGenesisBlock`:
//! - `nonce = 1`
//! - `extraData = SendRoot[:]` (32 zero bytes at genesis)
//! - `mixHash` = `[SendCount(8) | L1BlockNumber(8) | ArbOSFormatVersion(8) | flags(8)]` BE
//! - `gasLimit = 1 << 50`
//! - `baseFee = 100_000_000` (0.1 gwei)
//! - `difficulty = 1`
//! - `coinbase = 0x0`
//! Nitro IGNORES the corresponding fields from the genesis JSON and overwrites
//! them with these constants, so arbreth must produce identical fields to
//! match Nitro's block 0 hash.

use alloy_primitives::{address, hex, Address, Bytes, B256, B64, U256};
use arb_node::chainspec::ArbChainSpecParser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;

fn build_chain_json(arbos_version: u64) -> String {
    serde_json::json!({
        "config": {
            "chainId": 421614,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "arbitrum": {
                "EnableArbOS": true,
                "AllowDebugPrecompiles": true,
                "DataAvailabilityCommittee": false,
                "InitialArbOSVersion": arbos_version,
                "InitialChainOwner": "0x0000000000000000000000000000000000000000",
                "GenesisBlockNum": 0u64,
            },
        },
        // Intentionally garbage values: the parser should overwrite all of these
        // with the Nitro-canonical genesis header constants.
        "nonce": "0xdeadbeef",
        "timestamp": "0x0",
        "extraData": "0x",
        "gasLimit": "0x1234",
        "difficulty": "0x9",
        "mixHash": "0x1111111111111111111111111111111111111111111111111111111111111111",
        "coinbase": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        "baseFeePerGas": "0xabc",
        "alloc": {},
    })
    .to_string()
}

fn expected_mix_hash(arbos_version: u64) -> B256 {
    let mut mix = [0u8; 32];
    mix[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    B256::from(mix)
}

#[test]
fn genesis_header_matches_nitro_for_arbos_v50() {
    let json = build_chain_json(50);
    let spec = ArbChainSpecParser::parse(&json).expect("parse");
    let header = spec.genesis_header();

    assert_eq!(header.nonce, B64::from(1u64.to_be_bytes()), "nonce should be 1 (Nitro init message marker)");
    assert_eq!(header.gas_limit, 1u64 << 50, "gasLimit should be GethBlockGasLimit");
    assert_eq!(
        header.base_fee_per_gas,
        Some(100_000_000),
        "baseFee should be InitialBaseFeeWei (0.1 gwei)"
    );
    assert_eq!(header.difficulty, alloy_primitives::U256::from(1u64));
    assert_eq!(header.beneficiary, Address::ZERO);
    assert_eq!(
        header.extra_data,
        Bytes::from(vec![0u8; 32]),
        "extraData should be 32 zero bytes (SendRoot at genesis)"
    );
    assert_eq!(
        header.mix_hash,
        expected_mix_hash(50),
        "mixHash should encode ArbOS version at bytes 16..24"
    );
}

#[test]
fn genesis_header_mix_hash_encodes_version_at_byte_23() {
    for &v in &[10u64, 11, 20, 30, 31, 32, 40, 41, 50, 51, 60] {
        let json = build_chain_json(v);
        let spec = ArbChainSpecParser::parse(&json).expect("parse");
        let header = spec.genesis_header();

        let mix = header.mix_hash;
        // byte 23 should hold the LSB of version (since version <= 0xFF for now)
        assert_eq!(mix[23], (v & 0xff) as u8, "version {v} byte 23 mismatch");
        // bytes 16..24 should be the big-endian u64 of version
        let mut expected_high = [0u8; 8];
        expected_high.copy_from_slice(&v.to_be_bytes());
        assert_eq!(&mix[16..24], &expected_high, "version {v} bytes 16..24 mismatch");
        // all other bytes must be zero at genesis
        for i in 0..16 {
            assert_eq!(mix[i], 0, "version {v} byte {i} should be zero");
        }
        for i in 24..32 {
            assert_eq!(mix[i], 0, "version {v} byte {i} should be zero");
        }
    }
}

#[test]
fn genesis_block_hash_is_deterministic_per_version() {
    // Two parses with the same chain config must produce the same hash.
    let json = build_chain_json(50);
    let spec_a = ArbChainSpecParser::parse(&json).expect("parse");
    let spec_b = ArbChainSpecParser::parse(&json).expect("parse");
    assert_eq!(spec_a.genesis_hash(), spec_b.genesis_hash());
}

#[test]
fn genesis_block_hash_varies_by_arbos_version() {
    // Different ArbOS versions must produce different genesis hashes
    // because mixHash differs. State root may or may not differ.
    let h50 = ArbChainSpecParser::parse(&build_chain_json(50))
        .expect("parse")
        .genesis_hash();
    let h60 = ArbChainSpecParser::parse(&build_chain_json(60))
        .expect("parse")
        .genesis_hash();
    assert_ne!(h50, h60, "different ArbOS versions should hash differently");
}

#[test]
fn parser_overrides_garbage_header_fields() {
    // Even with deliberately-wrong header fields in the input JSON, the parsed
    // header must use the Nitro-canonical constants. This is the safety net
    // protecting Verify mode from producing junk hashes when fixtures get
    // header fields wrong.
    let mut json = serde_json::from_str::<serde_json::Value>(&build_chain_json(50)).unwrap();
    // Make the fields even more obviously wrong.
    json["nonce"] = serde_json::json!("0xffffffff");
    json["mixHash"] = serde_json::json!("0xabababababababababababababababababababababababababababababababab");
    json["gasLimit"] = serde_json::json!("0x1");

    let spec = ArbChainSpecParser::parse(&json.to_string()).expect("parse");
    let header = spec.genesis_header();
    assert_eq!(header.nonce, B64::from(1u64.to_be_bytes()));
    assert_eq!(header.mix_hash, expected_mix_hash(50));
    assert_eq!(header.gas_limit, 1u64 << 50);
}

#[test]
fn genesis_state_root_is_non_empty_with_arbos_alloc() {
    // The injected ArbOS alloc must produce a non-empty state root, otherwise
    // we wouldn't have any ArbOS state at all.
    let json = build_chain_json(50);
    let spec = ArbChainSpecParser::parse(&json).expect("parse");
    let header = spec.genesis_header();
    let empty_root: B256 = hex!("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421").into();
    assert_ne!(header.state_root, empty_root, "state root should not be empty");
}

#[test]
fn parser_injects_sentinels_without_skip_flag() {
    let json = build_chain_json(50);
    let spec = ArbChainSpecParser::parse(&json).expect("parse");

    let arbsys = address!("0000000000000000000000000000000000000064");
    assert!(
        spec.genesis().alloc.contains_key(&arbsys),
        "alloc should contain the ArbSys sentinel without SkipGenesisInjection"
    );
}

#[test]
fn parser_honors_skip_genesis_injection_flag() {
    let user_addr: Address = address!("00000000000000000000000000000000deadbeef");
    let user_balance = U256::from(0x1234u64);

    let json = serde_json::json!({
        "config": {
            "chainId": 421614,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "arbitrum": {
                "EnableArbOS": true,
                "InitialArbOSVersion": 50u64,
                "InitialChainOwner": "0x0000000000000000000000000000000000000000",
                "GenesisBlockNum": 0u64,
                "SkipGenesisInjection": true,
            },
        },
        "nonce": "0x1",
        "timestamp": "0x0",
        "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "gasLimit": "0x4000000000000",
        "difficulty": "0x1",
        "mixHash": "0x00000000000000000000000000000000000000000000003200000000000000000",
        "coinbase": "0x0000000000000000000000000000000000000000",
        "baseFeePerGas": "0x5f5e100",
        "alloc": {
            "00000000000000000000000000000000deadbeef": {
                "balance": "0x1234"
            }
        }
    })
    .to_string();

    let spec = ArbChainSpecParser::parse(&json).expect("parse");
    let alloc = &spec.genesis().alloc;

    assert_eq!(alloc.len(), 1, "alloc must contain only the supplied entry");
    let entry = alloc.get(&user_addr).expect("supplied entry preserved");
    assert_eq!(entry.balance, user_balance);

    // Confirm none of the standard precompile sentinels were injected.
    let arbsys = address!("0000000000000000000000000000000000000064");
    let arb_owner = address!("0000000000000000000000000000000000000070");
    assert!(!alloc.contains_key(&arbsys));
    assert!(!alloc.contains_key(&arb_owner));
}

#[test]
fn parser_skips_override_when_no_arbos_version() {
    // For chain configs without `arbitrum.InitialArbOSVersion`, the parser
    // must not touch the header. This preserves backward compatibility with
    // pre-curated allocs (e.g. the canonical Arbitrum Sepolia genesis JSON
    // pinned in the repo).
    let json = serde_json::json!({
        "config": {
            "chainId": 421614,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "terminalTotalDifficulty": 0,
            "terminalTotalDifficultyPassed": true
        },
        "nonce": "0x0000000000000001",
        "timestamp": "0x0",
        "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "gasLimit": "0x4000000000000",
        "difficulty": "0x1",
        "mixHash": "0x00000000000000000000000000000000000000000000000a0000000000000000",
        "coinbase": "0x0000000000000000000000000000000000000000",
        "baseFeePerGas": "0x5f5e100",
        "alloc": {}
    })
    .to_string();
    let spec = ArbChainSpecParser::parse(&json).expect("parse");
    let header = spec.genesis_header();
    // Pinned-alloc chains with v=10 should still report mix_hash as encoded for v=10
    let mut v10_mix = [0u8; 32];
    v10_mix[16..24].copy_from_slice(&10u64.to_be_bytes());
    assert_eq!(header.mix_hash, B256::from(v10_mix));
    assert_eq!(header.nonce, B64::from(1u64.to_be_bytes()));
    assert_eq!(header.gas_limit, 1u64 << 50);
}
