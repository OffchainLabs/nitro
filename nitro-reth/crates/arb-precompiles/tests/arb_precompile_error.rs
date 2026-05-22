use alloy_primitives::Bytes;
use arb_precompiles::ArbPrecompileError;
use arb_storage_errors::{DatabaseError, StorageError};
use revm::precompile::PrecompileError;

#[test]
fn revert_round_trips_through_revm_mapper() {
    let payload = Bytes::from_static(b"reason");
    let selector = [0x12u8, 0x34, 0x56, 0x78];
    let err = ArbPrecompileError::Revert {
        selector: Some(selector),
        data: payload.clone(),
        gas_used: 123,
    };

    let output = err.into_precompile_result(10_000).expect("ok-revert");
    assert!(output.reverted);
    assert_eq!(output.gas_used, 123);
    let bytes = output.bytes.as_ref();
    assert_eq!(&bytes[..4], &selector);
    assert_eq!(&bytes[4..], payload.as_ref());
}

#[test]
fn revert_caps_gas_used_at_gas_limit() {
    let err = ArbPrecompileError::Revert {
        selector: None,
        data: Bytes::new(),
        gas_used: 5_000,
    };
    let output = err.into_precompile_result(1_000).expect("ok-revert");
    assert!(output.reverted);
    assert_eq!(output.gas_used, 1_000);
}

#[test]
fn out_of_gas_maps_to_revm_out_of_gas() {
    let revm_err: PrecompileError = ArbPrecompileError::OutOfGas.into();
    assert!(matches!(revm_err, PrecompileError::OutOfGas));
}

#[test]
fn unsupported_method_propagates_as_other() {
    let revm_err: PrecompileError = ArbPrecompileError::UnsupportedMethod {
        method_id: [0xaa, 0xbb, 0xcc, 0xdd],
        required_version: 60,
    }
    .into();
    assert!(matches!(revm_err, PrecompileError::Other(_)));
}

#[test]
fn fatal_propagates_source_via_display() {
    #[derive(Debug, thiserror::Error)]
    #[error("disk on fire: {0}")]
    struct DiskFire(&'static str);

    let err = ArbPrecompileError::fatal(DiskFire("hot"));
    let display = format!("{err}");
    assert!(display.contains("disk on fire: hot"));

    let revm_err: PrecompileError = err.into();
    match revm_err {
        PrecompileError::Fatal(msg) => assert!(msg.contains("disk on fire: hot")),
        other => panic!("expected Fatal, got {other:?}"),
    }
}

#[test]
fn storage_error_converts_to_fatal() {
    let storage_err = StorageError::Invariant("queue read past next_put");
    let arb_err: ArbPrecompileError = storage_err.into();
    assert!(matches!(arb_err, ArbPrecompileError::Fatal(_)));
    let display = format!("{arb_err}");
    assert!(display.contains("queue read past next_put"));
}

#[test]
fn database_error_chain_surfaces_in_fatal_display() {
    let db_err = DatabaseError::custom(std::io::Error::other("db gone"));
    let arb_err: ArbPrecompileError = StorageError::Database(db_err).into();
    let display = format!("{arb_err}");
    assert!(display.contains("db gone"));
}
