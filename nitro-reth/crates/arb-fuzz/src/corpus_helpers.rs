use std::{
    collections::hash_map::DefaultHasher,
    hash::{Hash, Hasher},
    path::{Path, PathBuf},
};

use arb_test_harness::DiffReport;
use serde::Serialize;

fn short_hash(json: &str) -> String {
    let mut hasher = DefaultHasher::new();
    json.hash(&mut hasher);
    format!("{:016x}", hasher.finish())
}

fn captured_dir() -> PathBuf {
    let manifest = env!("CARGO_MANIFEST_DIR");
    let crates_root = Path::new(manifest)
        .parent()
        .map(Path::to_path_buf)
        .unwrap_or_else(|| PathBuf::from("crates"));
    crates_root
        .join("arb-spec-tests")
        .join("fixtures")
        .join("_captured")
}

#[derive(Serialize)]
struct CrashRecord<'a, I: Serialize> {
    input: &'a I,
    report: &'a DiffReport,
}

/// Write `(input, report)` as a fixture JSON; falls back to `temp_dir` on I/O failure.
pub fn dump_crash_as_fixture<I: Serialize>(input: &I, report: &DiffReport) -> PathBuf {
    let record = CrashRecord { input, report };
    let json = match serde_json::to_string_pretty(&record) {
        Ok(s) => s,
        Err(_) => "{\"input\":null,\"report\":null}".to_string(),
    };
    let hash = short_hash(&json);
    let filename = format!("fuzz_crash_{hash}.json");

    let primary = captured_dir();
    if std::fs::create_dir_all(&primary).is_ok() {
        let path = primary.join(&filename);
        if std::fs::write(&path, &json).is_ok() {
            return path;
        }
    }

    let fallback = std::env::temp_dir().join(filename);
    let _ = std::fs::write(&fallback, &json);
    fallback
}
