use std::path::{Path, PathBuf};

use walkdir::WalkDir;

use crate::{
    case::{SpecCase, SpecError},
    execution::ExecutionFixture,
};

pub const RPC_URL_ENV: &str = "ARB_SPEC_RPC_URL";

pub fn run_fixture(path: &Path) -> Result<(), SpecError> {
    let case = SpecCase::load(path)?;
    case.run().map_err(|e| match e {
        SpecError::Assertion(msg) => SpecError::Assertion(format!("{}: {msg}", path.display())),
        other => other,
    })
}

pub fn run_execution_fixture(path: &Path, rpc_url: &str) -> Result<(), SpecError> {
    let fixture = ExecutionFixture::load(path)?;
    fixture.run(rpc_url).map_err(|e| match e {
        SpecError::Assertion(msg) => SpecError::Assertion(format!("{}: {msg}", path.display())),
        other => other,
    })
}

pub fn run_execution_dir(dir: &Path) {
    let Ok(rpc_url) = std::env::var(RPC_URL_ENV) else {
        eprintln!(
            "skipping execution fixtures under {}: {RPC_URL_ENV} not set",
            dir.display()
        );
        return;
    };
    assert!(dir.exists(), "fixture dir missing: {}", dir.display());
    let mut failures = Vec::new();
    let mut count = 0;
    for entry in WalkDir::new(dir).into_iter().filter_map(Result::ok) {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        count += 1;
        if let Err(e) = run_execution_fixture(path, &rpc_url) {
            failures.push(format!("{}: {e}", path.display()));
        }
    }
    assert!(count > 0, "no fixtures found under {}", dir.display());
    if !failures.is_empty() {
        panic!(
            "{}/{} execution fixtures failed:\n  {}",
            failures.len(),
            count,
            failures.join("\n  ")
        );
    }
}

pub fn run_dir(dir: &Path) {
    assert!(dir.exists(), "fixture dir missing: {}", dir.display());
    let mut failures = Vec::new();
    let mut count = 0;
    for entry in WalkDir::new(dir).into_iter().filter_map(Result::ok) {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        count += 1;
        if let Err(e) = run_fixture(path) {
            failures.push(format!("{}: {e}", path.display()));
        }
    }
    assert!(count > 0, "no fixtures found under {}", dir.display());
    if !failures.is_empty() {
        panic!(
            "{}/{} fixtures failed:\n  {}",
            failures.len(),
            count,
            failures.join("\n  ")
        );
    }
}

pub fn fixtures_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("fixtures")
}
