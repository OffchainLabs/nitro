use std::collections::BTreeMap;
use std::fs;
use std::process::ExitCode;

use anyhow::{Context, Result};

use super::workspace_root;

pub fn run() -> Result<ExitCode> {
    let root = workspace_root();
    let lib = root.join("crates/arb-chainspec/src/lib.rs");
    let source = fs::read_to_string(&lib)
        .with_context(|| format!("read {}", lib.display()))?;

    let versions = parse_arbos_versions(&source);
    if versions.is_empty() {
        eprintln!("no ARBOS_VERSION_* constants parsed from {}", lib.display());
        return Ok(ExitCode::from(1));
    }

    let fixtures_dir = root.join("crates/arb-spec-tests/fixtures/arbos");
    let mut missing: Vec<u64> = Vec::new();
    let mut present: Vec<u64> = Vec::new();

    println!("ArbOS version baseline coverage:");
    for (version, names) in &versions {
        let path = fixtures_dir.join(format!("v{version}_baseline.json"));
        let exists = path.exists();
        let label = names.join(", ");
        if exists {
            present.push(*version);
            println!("  [OK]      v{version:<3} ({label}) -> {}", relative(&path, &root));
        } else {
            missing.push(*version);
            println!(
                "  [MISSING] v{version:<3} ({label}) -> {}",
                relative(&path, &root)
            );
        }
    }

    println!();
    println!(
        "summary: {present} present, {missing} missing (of {total} numeric ArbOS versions)",
        present = present.len(),
        missing = missing.len(),
        total = versions.len()
    );

    if missing.is_empty() {
        Ok(ExitCode::SUCCESS)
    } else {
        eprintln!(
            "fail: missing baseline fixtures for ArbOS versions: {:?}",
            missing
        );
        Ok(ExitCode::from(1))
    }
}

/// Parse `pub const ARBOS_VERSION_<NAME>: u64 = <RHS>;` declarations into a map keyed by numeric
/// version. RHS may be a literal or another `ARBOS_VERSION_*` alias which is followed transitively.
fn parse_arbos_versions(source: &str) -> BTreeMap<u64, Vec<String>> {
    let mut numeric: BTreeMap<String, u64> = BTreeMap::new();
    let mut aliases: BTreeMap<String, String> = BTreeMap::new();

    for line in source.lines() {
        let trimmed = line.trim();
        let Some(rest) = trimmed.strip_prefix("pub const ARBOS_VERSION_") else {
            continue;
        };
        let Some(colon) = rest.find(':') else { continue };
        let name = format!("ARBOS_VERSION_{}", &rest[..colon]);
        let after_colon = &rest[colon + 1..];
        let Some(eq) = after_colon.find('=') else { continue };
        let rhs = after_colon[eq + 1..]
            .trim()
            .trim_end_matches(';')
            .trim();
        if let Ok(n) = rhs.parse::<u64>() {
            numeric.insert(name, n);
        } else if let Some(alias) = rhs.strip_prefix("ARBOS_VERSION_").map(|s| format!("ARBOS_VERSION_{s}")) {
            aliases.insert(name, alias);
        }
    }

    let mut resolved: BTreeMap<String, u64> = numeric.clone();
    for (alias_name, target) in &aliases {
        let mut cursor = target.clone();
        let mut steps = 0usize;
        let max_steps = aliases.len() + 1;
        loop {
            if let Some(n) = numeric.get(&cursor) {
                resolved.insert(alias_name.clone(), *n);
                break;
            }
            if let Some(next) = aliases.get(&cursor) {
                cursor = next.clone();
                steps += 1;
                if steps > max_steps {
                    break;
                }
                continue;
            }
            break;
        }
    }

    let mut by_version: BTreeMap<u64, Vec<String>> = BTreeMap::new();
    for (name, version) in resolved {
        by_version.entry(version).or_default().push(name);
    }
    for names in by_version.values_mut() {
        names.sort();
    }
    by_version
}

fn relative(path: &std::path::Path, root: &std::path::Path) -> String {
    path.strip_prefix(root)
        .map(|p| p.display().to_string())
        .unwrap_or_else(|_| path.display().to_string())
}
