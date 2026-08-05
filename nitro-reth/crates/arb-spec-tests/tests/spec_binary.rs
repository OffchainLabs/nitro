// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//! MOCK (NIT-5205): validates the spec-test binary plumbing (release build →
//! artifact upload → download → chmod → env wiring) end to end until the
//! real spec suite is migrated. Compiled only with --features spec-binary,
//! so plain workspace test runs don't require the binary.
#![cfg(feature = "spec-binary")]

#[test]
fn spec_binary_runs() {
    if std::env::var("ARB_SPEC_REQUIRE_BINARY").as_deref() != Ok("1") {
        eprintln!("ARB_SPEC_REQUIRE_BINARY != 1; skipping binary check");
        return;
    }
    let binary = std::env::var("ARB_SPEC_BINARY")
        .expect("ARB_SPEC_BINARY must be set when ARB_SPEC_REQUIRE_BINARY=1");
    let output = std::process::Command::new(&binary)
        .arg("--version")
        .output()
        .unwrap_or_else(|err| panic!("failed to run {binary}: {err}"));
    assert!(
        output.status.success(),
        "{binary} --version exited with {:?}: {}",
        output.status,
        String::from_utf8_lossy(&output.stderr),
    );
}
