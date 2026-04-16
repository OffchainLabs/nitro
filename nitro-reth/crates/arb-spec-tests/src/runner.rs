use std::path::{Path, PathBuf};

use walkdir::WalkDir;

use crate::case::{SpecCase, SpecError};

pub fn run_fixture(path: &Path) -> Result<(), SpecError> {
    let case = SpecCase::load(path)?;
    case.run().map_err(|e| match e {
        SpecError::Assertion(msg) => SpecError::Assertion(format!("{}: {msg}", path.display())),
        other => other,
    })
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
