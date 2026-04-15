//! Ports of Nitro's `precompiles/constraints_test.go`.
//!
//! These tests validate the ArbOwner ↔ ArbGasInfo round-trip for
//! `setGasPricingConstraints` / `getGasPricingConstraints` and their
//! multi-gas siblings at ArbOS v50/v60. They drive the real owner-gated
//! setter (writing through our journaled state) and then feed the
//! mutated state back into the getter precompile to verify field-level
//! consistency, exactly like Nitro does against its in-memory
//! `arbosState`.
//!
//! Only the parts of Nitro's test file that are reachable with the
//! unit-test harness are ported. Tests that depend on `GasModelToUse`
//! flipping are intentionally skipped — we don't implement that field
//! yet and wiring it up without live evidence would risk a regression.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbgasinfo_precompile, create_arbowner_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
};
use common::{calldata, word_u64, PrecompileTest};

fn arbowner() -> DynPrecompile {
    create_arbowner_precompile()
}
fn arbgasinfo() -> DynPrecompile {
    create_arbgasinfo_precompile()
}

const OWNER: Address = address!("00000000000000000000000000000000000000aa");

fn chain_owner_member_slot(owner: Address) -> U256 {
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_as_b256 = B256::left_padding_from(owner.as_slice());
    map_slot_b256(by_address_key.as_slice(), &addr_as_b256)
}

fn install_owner(test: PrecompileTest, owner: Address) -> PrecompileTest {
    test.storage(
        ARBOS_STATE_ADDRESS,
        chain_owner_member_slot(owner),
        U256::from(1),
    )
}

/// Owner-authorized precompile test context at the given ArbOS version.
fn owner_fixture(arbos_version: u64) -> PrecompileTest {
    install_owner(
        PrecompileTest::new()
            .arbos_version(arbos_version)
            .caller(OWNER)
            .arbos_state(),
        OWNER,
    )
}

/// Build calldata for `setGasPricingConstraints(uint64[3][])`.
/// `uint64[3]` is a static inline tuple, so dynamic-array offset + length
/// are followed directly by three 32-byte words per element.
fn set_gas_pricing_calldata(constraints: &[[u64; 3]]) -> alloy_primitives::Bytes {
    let mut words: Vec<B256> = Vec::with_capacity(2 + 3 * constraints.len());
    words.push(word_u64(0x20)); // offset to the dynamic array
    words.push(word_u64(constraints.len() as u64)); // length
    for c in constraints {
        words.push(word_u64(c[0])); // target
        words.push(word_u64(c[1])); // adjustment window
        words.push(word_u64(c[2])); // backlog
    }
    calldata("setGasPricingConstraints(uint64[3][])", &words)
}

/// Build calldata for `setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])`.
///
/// Nitro's `MultiGasConstraint` struct is declared with
/// `Resources` (dynamic) FIRST, then `AdjustmentWindowSecs`,
/// `TargetPerSec`, `Backlog`. ABI encoding follows declaration order,
/// so each struct head starts with an offset-to-resources, then the
/// three static fields, and the resources array is appended in the
/// struct tail.
fn set_multi_gas_pricing_calldata(
    constraints: &[(Vec<(u8, u64)>, u32, u64, u64)],
) -> alloy_primitives::Bytes {
    // Each constraint head is 4 words (offset_to_resources, window, target, backlog).
    // Its resources tail = 1 length word + 2 words per (resource,weight).
    let mut buf = Vec::new();
    buf.extend_from_slice(&common::selector(
        "setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])",
    ));

    // Outer array offset (always 0x20 since it's the only top-level arg).
    let mut push_word = |buf: &mut Vec<u8>, v: U256| buf.extend_from_slice(&v.to_be_bytes::<32>());
    push_word(&mut buf, U256::from(32u64));
    push_word(&mut buf, U256::from(constraints.len() as u64));

    // Header of the outer array: one offset per struct element, relative
    // to the start of the "offsets area" (i.e. after selector+outer offset+length).
    let n = constraints.len();
    let mut struct_sizes: Vec<usize> = Vec::with_capacity(n);
    for c in constraints {
        let resources_len = c.0.len();
        // head(4*32) + tail(1*32 + resources_len*2*32)
        struct_sizes.push(4 * 32 + 32 + resources_len * 64);
    }
    let mut running = (n * 32) as u64;
    for size in &struct_sizes {
        push_word(&mut buf, U256::from(running));
        running += *size as u64;
    }

    // Struct bodies.
    for c in constraints {
        let resources = &c.0;
        let window = c.1;
        let target = c.2;
        let backlog = c.3;
        // Offset-to-resources relative to this struct's start = 4 head words.
        push_word(&mut buf, U256::from(4u64 * 32));
        push_word(&mut buf, U256::from(window));
        push_word(&mut buf, U256::from(target));
        push_word(&mut buf, U256::from(backlog));
        // Resources array: length + each (resource,weight) pair.
        push_word(&mut buf, U256::from(resources.len() as u64));
        for &(r, w) in resources {
            push_word(&mut buf, U256::from(r));
            push_word(&mut buf, U256::from(w));
        }
    }
    alloy_primitives::Bytes::from(buf)
}

/// Decode `MultiGasConstraint[]` as returned by `getMultiGasPricingConstraints()`.
/// Returns `(resources, window, target, backlog)` per struct in the outer array.
fn decode_multi_gas_pricing_constraints(out: &[u8]) -> Vec<(Vec<(u8, u64)>, u32, u64, u64)> {
    // Layout: outer offset(32) | length(32) | offset[](n) | struct bodies
    assert!(out.len() >= 64);
    let length = U256::from_be_slice(&out[32..64]).to::<u64>() as usize;
    let offsets_base = 64;
    let mut result = Vec::with_capacity(length);
    for i in 0..length {
        let offset_pos = offsets_base + i * 32;
        let offset = U256::from_be_slice(&out[offset_pos..offset_pos + 32]).to::<u64>() as usize;
        // struct start is relative to the start of the offsets area
        let struct_start = offsets_base + offset;
        let window =
            U256::from_be_slice(&out[struct_start + 32..struct_start + 64]).to::<u64>() as u32;
        let target = U256::from_be_slice(&out[struct_start + 64..struct_start + 96]).to::<u64>();
        let backlog = U256::from_be_slice(&out[struct_start + 96..struct_start + 128]).to::<u64>();
        let resources_offset =
            U256::from_be_slice(&out[struct_start..struct_start + 32]).to::<u64>() as usize;
        let resources_start = struct_start + resources_offset;
        let num_resources =
            U256::from_be_slice(&out[resources_start..resources_start + 32]).to::<u64>() as usize;
        let mut resources = Vec::with_capacity(num_resources);
        for j in 0..num_resources {
            let r_start = resources_start + 32 + j * 64;
            let r = U256::from_be_slice(&out[r_start..r_start + 32]).to::<u64>() as u8;
            let w = U256::from_be_slice(&out[r_start + 32..r_start + 64]).to::<u64>();
            resources.push((r, w));
        }
        result.push((resources, window, target, backlog));
    }
    result
}

/// Decode the `uint64[3][]` return of `getGasPricingConstraints()`.
fn decode_gas_pricing_constraints(out: &[u8]) -> Vec<[u64; 3]> {
    // layout: offset(32) | length(32) | each elem 3*32
    assert!(out.len() >= 64);
    let length = U256::from_be_slice(&out[32..64]).to::<u64>() as usize;
    let mut result = Vec::with_capacity(length);
    for i in 0..length {
        let base = 64 + i * 96;
        let t = U256::from_be_slice(&out[base..base + 32]).to::<u64>();
        let w = U256::from_be_slice(&out[base + 32..base + 64]).to::<u64>();
        let b = U256::from_be_slice(&out[base + 64..base + 96]).to::<u64>();
        result.push([t, w, b]);
    }
    result
}

// ── Ports of constraints_test.go ───────────────────────────────────────

/// Port of Nitro's `TestFailToSetInvalidConstraints`.
/// Rejects zero target and zero adjustment window.
///
/// Note: at ArbOS >= 11, `gas_check` converts `Err(Other)` into
/// `Ok(reverted=true)` to mirror Nitro's behavior, so we inspect the
/// `reverted` flag rather than `Result::is_err`.
#[test]
fn nitro_parity_fail_to_set_invalid_constraints() {
    // Zero target.
    let run = owner_fixture(50).call(&arbowner(), &set_gas_pricing_calldata(&[[0, 17, 1000]]));
    let out = run.result.as_ref().expect("should return Ok(reverted)");
    assert!(out.reverted, "zero target should revert");

    // Zero adjustment window.
    let run = owner_fixture(50).call(
        &arbowner(),
        &set_gas_pricing_calldata(&[[10_000_000, 0, 0]]),
    );
    let out = run.result.as_ref().expect("should return Ok(reverted)");
    assert!(out.reverted, "zero adjustment window should revert");
}

/// Port of Nitro's `TestSetLegacyBacklog`.
/// Setter/getter round-trip for the legacy gas backlog field.
#[test]
fn nitro_parity_set_legacy_backlog_round_trip() {
    // Initially zero.
    let run = owner_fixture(50).call(&arbgasinfo(), &calldata("getGasBacklog()", &[]));
    assert_eq!(U256::from_be_slice(run.output()), U256::ZERO);

    // Set to 80_000.
    let run = owner_fixture(50).call(
        &arbowner(),
        &calldata("setGasBacklog(uint64)", &[word_u64(80_000)]),
    );
    let _ = run.assert_ok();

    // Read back through a fresh GetInfo call that inherits the setter's
    // storage mutations.
    let getter = run.continue_into(owner_fixture(50), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbgasinfo(), &calldata("getGasBacklog()", &[]));
    assert_eq!(U256::from_be_slice(run.output()), U256::from(80_000));
}

/// Port of Nitro's `TestConstraintsStorage` — first half.
/// Set two constraints, verify the getter returns them with field-level
/// fidelity.
#[test]
fn nitro_parity_constraints_storage_round_trip_two_constraints() {
    let constraints = [[30_000_000, 1, 800_000], [15_000_000, 102, 1_600_000]];
    let set_run = owner_fixture(50).call(&arbowner(), &set_gas_pricing_calldata(&constraints));
    let _ = set_run.assert_ok();

    let getter = set_run.continue_into(owner_fixture(50), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbgasinfo(), &calldata("getGasPricingConstraints()", &[]));
    let got = decode_gas_pricing_constraints(run.output());
    assert_eq!(got.len(), 2);
    assert_eq!(got[0], [30_000_000, 1, 800_000]);
    assert_eq!(got[1], [15_000_000, 102, 1_600_000]);
}

/// Port of Nitro's `TestConstraintsBacklogUpdate`. Backlog is the only
/// mutable field after `SetGasPricingConstraints`, so a subsequent state
/// update must be observable through the getter.
///
/// We can't call `state.L2PricingState().OpenGasConstraintAt(i).SetBacklog()`
/// from the test (no ArbosState bindings in Rust), so we mutate the raw
/// slot directly in the continued fixture. Slot layout is `vector_element_field(vec, i,
/// CONSTRAINT_BACKLOG=2)`.
#[test]
fn nitro_parity_constraints_backlog_update() {
    use arb_precompiles::storage_slot::{gas_constraints_vec_key, vector_element_field};

    let set_run = owner_fixture(50).call(
        &arbowner(),
        &set_gas_pricing_calldata(&[[30_000_000, 1, 0], [15_000_000, 86400, 8000]]),
    );
    let _ = set_run.assert_ok();

    // Overwrite the backlog field for both constraints.
    let vec_key = gas_constraints_vec_key();
    let backlog_slot_0 = vector_element_field(&vec_key, 0, 2);
    let backlog_slot_1 = vector_element_field(&vec_key, 1, 2);

    let base = set_run
        .continue_into(owner_fixture(50), ARBOS_STATE_ADDRESS)
        .storage(
            ARBOS_STATE_ADDRESS,
            backlog_slot_0,
            U256::from(5_000_000_u64),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            backlog_slot_1,
            U256::from(10_000_000_u64),
        );

    let run = base.call(&arbgasinfo(), &calldata("getGasPricingConstraints()", &[]));
    let got = decode_gas_pricing_constraints(run.output());
    assert_eq!(got.len(), 2);
    assert_eq!(
        got[0][2], 5_000_000,
        "backlog for element 0 must reflect update"
    );
    assert_eq!(
        got[1][2], 10_000_000,
        "backlog for element 1 must reflect update"
    );
}

/// Port of Nitro's `TestMultiGasConstraintsCantExceedLimit`. A single
/// constraint with backlog high enough to push the pricing exponent
/// past `MaxPricingExponentBips` must revert.
#[test]
fn nitro_parity_multi_gas_constraints_cant_exceed_limit() {
    let run = owner_fixture(60).call(
        &arbowner(),
        &set_multi_gas_pricing_calldata(&[(
            vec![(0, 1), (1, 2)], // Computation=1, StorageAccess=2
            1,                    // adjustment window secs
            30_000_000,           // target per sec
            800_000_000_000,      // backlog — intentionally huge
        )]),
    );
    let out = run.result.as_ref().expect("should return Ok(reverted)");
    assert!(
        out.reverted,
        "backlog that exceeds MaxPricingExponentBips must revert"
    );
}

/// Port of Nitro's `TestGetMultiGasPricingConstraintsOrder`. Verifies
/// that `getMultiGasPricingConstraints()` returns each constraint's
/// `Resources` list sorted ascending by resource-kind id, regardless of
/// the order the caller supplied them.
#[test]
fn nitro_parity_multi_gas_pricing_constraints_order() {
    // Supply resources in deliberately unsorted order.
    let constraints = vec![(
        vec![(3u8, 7u64), (0u8, 5u64), (2u8, 3u64), (1u8, 1u64)],
        1u32,
        20_000_000u64,
        800_000u64,
    )];
    let set_run =
        owner_fixture(60).call(&arbowner(), &set_multi_gas_pricing_calldata(&constraints));
    let _ = set_run.assert_ok();

    let getter = set_run.continue_into(owner_fixture(60), ARBOS_STATE_ADDRESS);
    let run = getter.call(
        &arbgasinfo(),
        &calldata("getMultiGasPricingConstraints()", &[]),
    );
    let got = decode_multi_gas_pricing_constraints(run.output());

    assert_eq!(got.len(), 1);
    let (resources, _, _, _) = &got[0];
    let kinds: Vec<u8> = resources.iter().map(|(k, _)| *k).collect();
    for i in 1..kinds.len() {
        assert!(
            kinds[i - 1] < kinds[i],
            "resources must be sorted by resource kind: got {kinds:?}"
        );
    }
}

/// Port of Nitro's `TestMultiGasConstraintsStorage`. Set two
/// multi-gas constraints with distinct resource weights and verify the
/// getter returns them field-by-field.
#[test]
fn nitro_parity_multi_gas_constraints_storage_round_trip() {
    let constraints = vec![
        (
            vec![(0u8, 1u64), (1u8, 2u64)],
            1u32,
            30_000_000u64,
            800_000u64,
        ),
        (
            vec![(0u8, 2u64), (1u8, 3u64)],
            102u32,
            15_000_000u64,
            1_600_000u64,
        ),
    ];
    let set_run =
        owner_fixture(60).call(&arbowner(), &set_multi_gas_pricing_calldata(&constraints));
    let out = set_run
        .result
        .as_ref()
        .expect("setter should not hard-error");
    assert!(!out.reverted, "setter unexpectedly reverted: {:?}", out);

    let getter = set_run.continue_into(owner_fixture(60), ARBOS_STATE_ADDRESS);
    let run = getter.call(
        &arbgasinfo(),
        &calldata("getMultiGasPricingConstraints()", &[]),
    );
    let got = decode_multi_gas_pricing_constraints(run.output());

    assert_eq!(got.len(), 2, "two constraints must be returned");

    // Element 0
    let (resources0, window0, target0, backlog0) = &got[0];
    assert_eq!(*window0, 1);
    assert_eq!(*target0, 30_000_000);
    assert_eq!(*backlog0, 800_000);
    let mut r0 = resources0.clone();
    r0.sort_by_key(|x| x.0);
    assert_eq!(r0, vec![(0, 1), (1, 2)]);

    // Element 1
    let (resources1, window1, target1, backlog1) = &got[1];
    assert_eq!(*window1, 102);
    assert_eq!(*target1, 15_000_000);
    assert_eq!(*backlog1, 1_600_000);
    let mut r1 = resources1.clone();
    r1.sort_by_key(|x| x.0);
    assert_eq!(r1, vec![(0, 2), (1, 3)]);
}

/// Port of Nitro's `TestConstraintsStorage` — second half.
/// Replacing the constraint list must clear the old entries.
#[test]
fn nitro_parity_constraints_storage_replace_clears_old() {
    // Start with two, then replace with one.
    let first = owner_fixture(50).call(
        &arbowner(),
        &set_gas_pricing_calldata(&[[30_000_000, 1, 800_000], [15_000_000, 102, 1_600_000]]),
    );
    let _ = first.assert_ok();

    // Replace.
    let base = first.continue_into(owner_fixture(50), ARBOS_STATE_ADDRESS);
    let second = base.call(
        &arbowner(),
        &set_gas_pricing_calldata(&[[7_000_000, 12, 50_000_000]]),
    );
    let _ = second.assert_ok();

    // Verify only the new constraint remains.
    let getter = second.continue_into(owner_fixture(50), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbgasinfo(), &calldata("getGasPricingConstraints()", &[]));
    let got = decode_gas_pricing_constraints(run.output());
    assert_eq!(got.len(), 1, "old constraints must be cleared");
    assert_eq!(got[0], [7_000_000, 12, 50_000_000]);
}
