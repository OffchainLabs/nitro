use std::collections::BTreeMap;
use std::process::ExitCode;

use anyhow::{Context, Result};
use walkdir::WalkDir;

use super::workspace_root;

pub fn run() -> Result<ExitCode> {
    let root = workspace_root();
    let fixtures = root.join("crates/arb-spec-tests/fixtures");

    if !fixtures.is_dir() {
        anyhow::bail!("fixtures dir not found: {}", fixtures.display());
    }

    let mut by_category: BTreeMap<String, Vec<String>> = BTreeMap::new();

    for entry in WalkDir::new(&fixtures).min_depth(1).into_iter() {
        let entry = entry.context("walk fixtures")?;
        if !entry.file_type().is_file() {
            continue;
        }
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        let rel = match path.strip_prefix(&fixtures) {
            Ok(p) => p,
            Err(_) => continue,
        };
        let category = if rel.components().count() > 1 {
            rel.components()
                .next()
                .map(|c| c.as_os_str().to_string_lossy().into_owned())
                .unwrap_or_else(|| "<root>".to_string())
        } else {
            "<root>".to_string()
        };

        let stem = path
            .file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or("")
            .to_string();
        by_category.entry(category).or_default().push(stem);
    }

    let total: usize = by_category.values().map(|v| v.len()).sum();

    println!("# Fixture inventory");
    println!();
    println!("Total: {total} fixtures across {} categories", by_category.len());
    println!();
    println!("| Category | Count | Per-version |");
    println!("|---|---|---|");
    for (cat, files) in &by_category {
        let per_version = summarize_versions(files);
        println!(
            "| {cat} | {count} | {per_version} |",
            count = files.len(),
            per_version = per_version,
        );
    }

    Ok(ExitCode::SUCCESS)
}

fn summarize_versions(files: &[String]) -> String {
    let mut counts: BTreeMap<u64, usize> = BTreeMap::new();
    for f in files {
        for tok in extract_v_tokens(f) {
            *counts.entry(tok).or_default() += 1;
        }
    }
    if counts.is_empty() {
        return "—".to_string();
    }
    let parts: Vec<String> = counts
        .iter()
        .map(|(v, n)| format!("v{v}:{n}"))
        .collect();
    parts.join(", ")
}

fn extract_v_tokens(stem: &str) -> Vec<u64> {
    let bytes = stem.as_bytes();
    let mut out: Vec<u64> = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        let c = bytes[i];
        let prev_is_word = i > 0 && {
            let p = bytes[i - 1];
            p.is_ascii_alphanumeric() || p == b'_'
        };
        if (c == b'v' || c == b'V') && !prev_is_word {
            let mut j = i + 1;
            while j < bytes.len() && bytes[j].is_ascii_digit() {
                j += 1;
            }
            if j > i + 1 {
                let s = &stem[i + 1..j];
                if let Ok(n) = s.parse::<u64>() {
                    if !out.contains(&n) {
                        out.push(n);
                    }
                }
                i = j;
                continue;
            }
        }
        i += 1;
    }
    out.sort();
    out
}
