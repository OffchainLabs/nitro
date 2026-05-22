//! Targeted parity tests pinning specific failure modes to the matching
//! [`StylusError`] variant.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use arb_stylus::{decompress_wasm, strip_stylus_prefix, StylusError};

#[test]
fn strip_stylus_prefix_rejects_short_bytecode() {
    let err = strip_stylus_prefix(&[0xEF, 0xF0]).expect_err("short bytecode must fail");
    assert!(
        matches!(err, StylusError::InvalidProgram(reason) if reason.contains("too short")),
        "expected InvalidProgram with too-short reason, got {err:?}",
    );
}

#[test]
fn strip_stylus_prefix_rejects_wrong_discriminant() {
    let err = strip_stylus_prefix(&[0x00, 0x00, 0x00, 0x00, 0x01])
        .expect_err("non-Stylus bytecode must fail");
    assert!(
        matches!(err, StylusError::InvalidProgram(reason) if reason.contains("discriminant")),
        "expected InvalidProgram with discriminant reason, got {err:?}",
    );
}

#[test]
fn decompress_wasm_rejects_non_stylus_bytecode() {
    let err = decompress_wasm(&[0x60, 0x00, 0x60, 0x00]).expect_err("non-Stylus must fail");
    assert!(
        matches!(err, StylusError::InvalidProgram(_)),
        "expected InvalidProgram, got {err:?}",
    );
}

#[test]
fn decompress_wasm_rejects_unknown_dictionary() {
    let err =
        decompress_wasm(&[0xEF, 0xF0, 0x00, 0x99, 0x00]).expect_err("unknown dictionary must fail");
    assert!(
        matches!(err, StylusError::InvalidProgram(reason) if reason.contains("dictionary")),
        "expected InvalidProgram with dictionary reason, got {err:?}",
    );
}

#[test]
fn out_of_ink_constructor_produces_correct_variant() {
    let err: StylusError = StylusError::out_of_ink::<()>().unwrap_err();
    assert!(matches!(err, StylusError::OutOfInk));
}

#[test]
fn internal_constructor_carries_message() {
    let err: StylusError = StylusError::internal::<()>("bridge failed").unwrap_err();
    match err {
        StylusError::Internal(msg) => assert_eq!(msg, "bridge failed"),
        other => panic!("expected Internal, got {other:?}"),
    }
}
