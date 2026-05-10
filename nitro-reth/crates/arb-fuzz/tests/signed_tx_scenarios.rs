//! Deterministic smoke tests for `DiffSignedTxScenario::into_scenario`.
//! Confirms the fuzz scenario constructor produces a well-formed
//! deposit + signed-tx pair for each supported tx kind, without needing
//! libfuzzer or Docker.

use alloy_primitives::Address;
use arb_fuzz::arbitrary_impls::{
    AuthInput, BoundedBytes, DiffSignedTxScenario, SignedTxKind,
};
use arb_test_harness::scenario::ScenarioStep;

fn arbos_v40() -> arb_fuzz::arbitrary_impls::ArbosVersion {
    use arbitrary::{Arbitrary, Unstructured};
    let mut data = vec![40u8; 32];
    let mut u = Unstructured::new(&mut data);
    arb_fuzz::arbitrary_impls::ArbosVersion::arbitrary(&mut u).unwrap()
}

fn fixed_to() -> Option<Address> {
    Some(Address::repeat_byte(0xab))
}

fn make(kind: SignedTxKind, with_to: bool, auths: Vec<AuthInput>) -> DiffSignedTxScenario {
    DiffSignedTxScenario {
        arbos_version: arbos_v40(),
        kind,
        signing_key_low: [7u8; 32],
        to: if with_to { fixed_to() } else { None },
        data: BoundedBytes::default(),
        value_low: 0,
        gas: 200_000,
        max_fee: 1_000_000_000,
        max_priority_fee: 100_000_000,
        authorizations: auths,
    }
}

fn make_auth() -> AuthInput {
    AuthInput {
        signing_key: [3u8; 32],
        address: Address::repeat_byte(0xcd),
        nonce: 0,
    }
}

fn count_messages(steps: &[ScenarioStep]) -> usize {
    steps
        .iter()
        .filter(|s| matches!(s, ScenarioStep::Message { .. }))
        .count()
}

#[test]
fn legacy_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Legacy, true, Vec::new())
        .into_scenario()
        .expect("legacy scenario builds");
    assert_eq!(count_messages(&s.steps), 2, "expected deposit + signed tx");
}

#[test]
fn eip2930_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Eip2930, true, Vec::new())
        .into_scenario()
        .expect("eip2930 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip1559_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Eip1559, true, Vec::new())
        .into_scenario()
        .expect("eip1559 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip7702_with_one_auth_builds() {
    let s = make(SignedTxKind::Eip7702, true, vec![make_auth()])
        .into_scenario()
        .expect("eip7702 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip7702_create_returns_none() {
    // EIP-7702 cannot be CREATE per spec.
    let scen = make(SignedTxKind::Eip7702, false, vec![make_auth()]);
    assert!(scen.into_scenario().is_none());
}

#[test]
fn eip7702_empty_auth_list_skipped_at_signed_step() {
    // Empty auth list makes SignedL2TxBuilder::build_envelope fail; the
    // scenario still builds (deposit succeeds) but only emits 1 message.
    let scen = make(SignedTxKind::Eip7702, true, Vec::new())
        .into_scenario()
        .expect("scenario still builds with deposit");
    assert_eq!(count_messages(&scen.steps), 1);
}
