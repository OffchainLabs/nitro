mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbwasmcache_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CACHE_MANAGERS_KEY,
        CHAIN_OWNER_SUBSPACE, PROGRAMS_DATA_KEY, PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, word_address, PrecompileTest};
use revm::state::AccountInfo;

fn arbwasmcache() -> DynPrecompile {
    create_arbwasmcache_precompile()
}

fn cache_manager_member_slot(addr: Address) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let cm_key = derive_subspace_key(programs_key.as_slice(), CACHE_MANAGERS_KEY);
    let by_addr_key = derive_subspace_key(cm_key.as_slice(), &[0]);
    let mut padded = [0u8; 32];
    padded[12..32].copy_from_slice(addr.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &B256::from(padded))
}

#[test]
fn is_cache_manager_returns_false_for_unregistered() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbwasmcache(),
        &calldata("isCacheManager(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn is_cache_manager_returns_true_for_member() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            cache_manager_member_slot(probe),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("isCacheManager(address)", &[word_address(probe)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn codehash_is_cached_reads_program_word_byte_14() {
    let codehash = B256::from([0x42; 32]);
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let slot = map_slot_b256(data_key.as_slice(), &codehash);
    let mut word = [0u8; 32];
    word[14] = 1;
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, slot, U256::from_be_bytes(word))
        .call(
            &arbwasmcache(),
            &calldata("codehashIsCached(bytes32)", &[codehash]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn codehash_is_cached_returns_false_when_byte_14_zero() {
    let codehash = B256::from([0x55; 32]);
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbwasmcache(),
        &calldata("codehashIsCached(bytes32)", &[codehash]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

// ── set_program_cached path tests ────────────────────────────────────

const ARBITRUM_START_TIME: u64 = 1_421_388_000;

fn hours_since_start(time: u64) -> u32 {
    ((time.saturating_sub(ARBITRUM_START_TIME)) / 3600) as u32
}

fn programs_params_slot() -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    map_slot(params_key.as_slice(), 0)
}

fn program_slot(codehash: B256) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    map_slot_b256(data_key.as_slice(), &codehash)
}

fn chain_owner_member_slot(addr: Address) -> U256 {
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_addr_key = derive_subspace_key(owner_key.as_slice(), &[0]);
    let mut padded = [0u8; 32];
    padded[12..32].copy_from_slice(addr.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &B256::from(padded))
}

/// Pack the StylusParams word with version and expiry_days, leaving everything
/// else zero. Mirrors the layout in arbos/programs/params.go::Save.
fn pack_params(version: u16, expiry_days: u16) -> U256 {
    let mut buf = [0u8; 32];
    buf[0..2].copy_from_slice(&version.to_be_bytes());
    buf[19..21].copy_from_slice(&expiry_days.to_be_bytes());
    U256::from_be_bytes(buf)
}

/// Pack a Program word matching arbos/programs/programs.go::setProgram.
fn pack_program(version: u16, activated_at_hours: u32, cached: bool) -> U256 {
    let mut buf = [0u8; 32];
    buf[0..2].copy_from_slice(&version.to_be_bytes());
    buf[8] = (activated_at_hours >> 16) as u8;
    buf[9] = (activated_at_hours >> 8) as u8;
    buf[10] = activated_at_hours as u8;
    buf[14] = if cached { 1 } else { 0 };
    U256::from_be_bytes(buf)
}

#[test]
fn cache_codehash_rejects_non_manager_non_owner() {
    // Caller is neither in cache managers nor in chain owners.
    let codehash = B256::from([0x11; 32]);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .gas(50_000)
        .call(
            &arbwasmcache(),
            &calldata("cacheCodehash(bytes32)", &[codehash]),
        );
    let out = run.assert_ok();
    assert!(out.reverted, "non-authorized cacheCodehash must revert");
}

#[test]
fn cache_codehash_succeeds_for_chain_owner_and_sets_cached_byte() {
    // Chain owner adds a fresh, in-version program to the cache.
    let codehash = B256::from([0x22; 32]);
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let now = 1_700_000_000_u64;
    let activated_hours = hours_since_start(now);
    let program_word = pack_program(2, activated_hours, false);

    let test = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(2, 365),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        );

    let run = test.call(
        &arbwasmcache(),
        &calldata("cacheCodehash(bytes32)", &[codehash]),
    );
    let _ = run.assert_ok();

    // Read the post-state program word and verify byte 14 flipped to 1.
    let stored = run.storage(ARBOS_STATE_ADDRESS, program_slot(codehash));
    let bytes = stored.to_be_bytes::<32>();
    assert_eq!(bytes[14], 1, "cached byte must be set to 1");
}

#[test]
fn cache_codehash_no_op_when_already_cached() {
    let codehash = B256::from([0x33; 32]);
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let now = 1_700_000_000_u64;
    let activated_hours = hours_since_start(now);
    let program_word = pack_program(2, activated_hours, true);

    let run = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(2, 365),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("cacheCodehash(bytes32)", &[codehash]),
        );
    let _ = run.assert_ok();
    let stored = run.storage(ARBOS_STATE_ADDRESS, program_slot(codehash));
    let bytes = stored.to_be_bytes::<32>();
    assert_eq!(bytes[14], 1, "still cached after no-op");
}

#[test]
fn cache_codehash_reverts_program_needs_upgrade_for_stale_version() {
    let codehash = B256::from([0x44; 32]);
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let now = 1_700_000_000_u64;
    let activated_hours = hours_since_start(now);
    // params at v3, program at v2 → ProgramNeedsUpgrade(2,3) when caching.
    let program_word = pack_program(2, activated_hours, false);

    let run = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(3, 365),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("cacheCodehash(bytes32)", &[codehash]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramNeedsUpgrade(uint16,uint16)");
    assert_eq!(&out.bytes[..4], &sel[..4]);
}

#[test]
fn cache_codehash_reverts_program_expired() {
    let codehash = B256::from([0x55; 32]);
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    // expiry_days = 1 → 86400s lifetime; program activated at hour 0; current = 2 days later.
    let now = ARBITRUM_START_TIME + 2 * 86_400;
    let program_word = pack_program(2, 0, false);

    let run = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(2, 1),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("cacheCodehash(bytes32)", &[codehash]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramExpired(uint64)");
    assert_eq!(&out.bytes[..4], &sel[..4]);
}

#[test]
fn evict_codehash_clears_cached_byte() {
    // Eviction does NOT check version/expiry per Nitro: only the cache toggle.
    let codehash = B256::from([0x66; 32]);
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let now = 1_700_000_000_u64;
    let activated_hours = hours_since_start(now);
    let program_word = pack_program(2, activated_hours, true);

    let run = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(2, 365),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("evictCodehash(bytes32)", &[codehash]),
        );
    let _ = run.assert_ok();
    let stored = run.storage(ARBOS_STATE_ADDRESS, program_slot(codehash));
    let bytes = stored.to_be_bytes::<32>();
    assert_eq!(bytes[14], 0, "cached byte must be cleared after evict");
}

#[test]
fn cache_program_resolves_address_to_codehash() {
    let codehash = B256::from([0x77; 32]);
    let prog_addr: Address = address!("000000000000000000000000000000000000beef");
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let now = 1_700_000_000_u64;
    let activated_hours = hours_since_start(now);
    let program_word = pack_program(2, activated_hours, false);

    let run = PrecompileTest::new()
        .arbos_version(32)
        .block_timestamp(now)
        .caller(owner)
        .arbos_state()
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            programs_params_slot(),
            pack_params(2, 365),
        )
        .storage(ARBOS_STATE_ADDRESS, program_slot(codehash), program_word)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbwasmcache(),
            &calldata("cacheProgram(address)", &[word_address(prog_addr)]),
        );
    let _ = run.assert_ok();
    let stored = run.storage(ARBOS_STATE_ADDRESS, program_slot(codehash));
    assert_eq!(stored.to_be_bytes::<32>()[14], 1);
}
