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
fn parser_with_skip_genesis_injection_still_injects_arbos_state() {
    // SkipGenesisInjection used to disable the alloc-injection step entirely,
    // but the cache files produced by `arb-test genesis-capture` set the flag
    // and are missing both `FilteredTransactionsState` and the storage trie of
    // the ArbOS state account, so honoring the flag verbatim breaks the
    // genesis state root. The parser now always merges the injection table:
    // user-supplied accounts/fields/slots win on conflict, anything missing
    // gets filled in.
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

    let entry = alloc.get(&user_addr).expect("user-supplied entry preserved");
    assert_eq!(entry.balance, user_balance);

    let arbsys = address!("0000000000000000000000000000000000000064");
    let arb_owner = address!("0000000000000000000000000000000000000070");
    let arbos_state = address!("a4b05fffffffffffffffffffffffffffffffffff");
    assert!(alloc.contains_key(&arbsys), "ArbSys sentinel must be injected");
    assert!(alloc.contains_key(&arb_owner), "ArbOwner sentinel must be injected");
    let state_account = alloc
        .get(&arbos_state)
        .expect("ArbOS state account must be injected");
    let storage = state_account
        .storage
        .as_ref()
        .expect("ArbOS state account must carry storage slots");
    assert!(!storage.is_empty(), "ArbOS storage must be populated");
}

/// Pinning tests: parsing each captured Nitro genesis cache file plus the
/// parser's always-on injection step must produce the exact state root and
/// hash the spec fixtures pin. Regressions in storage layout, chain-config
/// serialization, FilteredTransactionsState seeding, or header overrides
/// will trip these long before the spec suite spins up the full binary.
fn assert_cache_state_root_matches(version: u64, expected_state_root: B256, expected_hash: B256) {
    let cache_path = format!(
        "{}/../arb-spec-tests/fixtures/_genesis_cache/chain412346_v{version}.json",
        env!("CARGO_MANIFEST_DIR")
    );
    let spec = ArbChainSpecParser::parse(&cache_path).expect("parse cache");
    let header = spec.genesis_header();
    assert_eq!(
        header.state_root, expected_state_root,
        "state root mismatch for v{version} cache"
    );
    assert_eq!(
        spec.genesis_hash(),
        expected_hash,
        "block hash mismatch for v{version} cache"
    );
}

#[test]
fn cache_state_root_matches_v10_baseline() {
    assert_cache_state_root_matches(
        10,
        hex!("5c0b434552d04dcd5c723669d1b332f1837dee591bb24a3aac637e3a2aca7185").into(),
        hex!("2116ae1892bcc05016222bd57af91945f425eeca1f7c6bacfcf0d67d4ed00f88").into(),
    );
}

#[test]
fn cache_state_root_matches_v32_baseline() {
    assert_cache_state_root_matches(
        32,
        hex!("471d193420070fdfecec116f46b15937d7244f595901ca12ef4ce4ddbf53cd68").into(),
        hex!("a6c2936f68ae2910e30976a234bb9f09aacb22b6dce5f3fe3a1b1c426b46fa57").into(),
    );
}

#[test]
fn cache_state_root_matches_v50_baseline() {
    assert_cache_state_root_matches(
        50,
        hex!("7df9d696ad8176680f675557ef0a0e9276d9510c040dc1a24a9296b4a9d8d694").into(),
        hex!("08cad158d0956351511d1e67cc48bc1ef089ec37c087d4e227b45a6fb71bac3e").into(),
    );
}

#[test]
fn cache_state_root_matches_v60_baseline() {
    assert_cache_state_root_matches(
        60,
        hex!("6f96ed9046ded46b40bd3ac903158065be54827c0624229e9c6e0ff541c0c6df").into(),
        hex!("d0564ff804de4b490ae5c3d252c66a886c10faa42730f983dd6bbe3dfbcfd07b").into(),
    );
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

#[test]
fn fresh_boot_v10_state_root_matches_nitro() {
    // Reproduces the Nitro Docker `--init.empty=true` boot for chain 421614
    // at ArbOS v10. The harness sends Nitro the chain-config JSON below, then
    // Nitro re-marshals it during init and writes it to the chain_config
    // subspace. This test checks the resulting state root and block hash
    // match what Nitro produces from that same input, including the
    // pre-fix FilteredTransactionsState account that older Nitro builds
    // unconditionally bump nonce on.
    let cfg = serde_json::json!({
        "config": {
            "chainId": 421614,
            "homesteadBlock": 0,
            "daoForkSupport": true,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "muirGlacierBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "depositContractAddress": "0x0000000000000000000000000000000000000000",
            "clique": {"period": 0, "epoch": 0},
            "arbitrum": {
                "EnableArbOS": true,
                "AllowDebugPrecompiles": false,
                "DataAvailabilityCommittee": false,
                "InitialArbOSVersion": 10,
                "InitialChainOwner": "0x71B61c2E250AFa05dFc36304D6c91501bE0965D8",
                "GenesisBlockNum": 0u64,
            }
        },
        "alloc": {}
    });
    let spec = ArbChainSpecParser::parse(&cfg.to_string()).expect("parse");
    let header = spec.genesis_header();
    let hash = spec.genesis_hash();

    let expected_state_root: B256 =
        hex!("ab6821d87dca1473891fee8b08d1582b61362bac1ce5bd7a6513afe6c86b1327").into();
    let expected_hash: B256 =
        hex!("c84425bb7ca6315b83ebcc96ca814b7b7fc7eab6a734c47b48c94195500414fa").into();
    assert_eq!(header.state_root, expected_state_root, "state root must match Nitro Docker fresh boot");
    assert_eq!(hash, expected_hash, "block hash must match Nitro Docker fresh boot");

    // Sanity: 16 accounts (13 v0 precompile sentinels + ArbosActs + ArbosState
    // + FilteredTransactionsState).
    assert_eq!(spec.genesis().alloc.len(), 16);

    let filtered_tx_state: Address = address!("a4b0500000000000000000000000000000000001");
    let acc = spec
        .genesis()
        .alloc
        .get(&filtered_tx_state)
        .expect("filtered tx state account must be present");
    assert_eq!(acc.nonce, Some(1));
    assert_eq!(acc.balance, U256::ZERO);
    assert!(acc.code.is_none() || acc.code.as_ref().unwrap().is_empty());
    assert!(acc.storage.is_none() || acc.storage.as_ref().unwrap().is_empty());
}

/// The curated `genesis/arbitrum-sepolia.json` is the production-Sepolia
/// boot file. Its config block has no `arbitrum.InitialArbOSVersion`, so
/// the parser's inject + override paths must stay disabled. The genesis
/// hash must equal what production Sepolia returns at block 0.
#[test]
fn production_sepolia_hash_unchanged() {
    let path = format!(
        "{}/../../genesis/arbitrum-sepolia.json",
        env!("CARGO_MANIFEST_DIR"),
    );
    let spec = ArbChainSpecParser::parse(&path).expect("parse production sepolia");
    let expected_hash: B256 =
        hex!("77194da4010e549a7028a9c3c51c3e277823be6ac7d138d0bb8a70197b5c004c").into();
    let expected_state_root: B256 =
        hex!("8647a2ae10b316ca12fbd76327fe4d64d12cb0ec664a128b0d59df15d05391be").into();
    assert_eq!(
        spec.genesis_header().state_root,
        expected_state_root,
        "production Sepolia state root must not regress"
    );
    assert_eq!(
        spec.genesis_hash(),
        expected_hash,
        "production Sepolia block hash must not regress"
    );
}
