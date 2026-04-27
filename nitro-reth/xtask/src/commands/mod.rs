pub mod check_coverage;
pub mod count_fixtures;
pub mod regen_inventory;

use std::path::PathBuf;

pub fn workspace_root() -> PathBuf {
    let manifest = std::env::var("CARGO_MANIFEST_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|_| std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")));
    manifest
        .parent()
        .map(PathBuf::from)
        .unwrap_or(manifest)
}
