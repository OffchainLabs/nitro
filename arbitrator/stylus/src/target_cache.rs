// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use eyre::{eyre, OptionExt, Result};
use lazy_static::lazy_static;
use parking_lot::RwLock;
use std::{
    collections::HashMap,
    str::FromStr,
    sync::atomic::{AtomicUsize, Ordering},
};
use wasmer_types::{CpuFeature, Target, Triple};

lazy_static! {
    static ref TARGET_CACHE: RwLock<HashMap<String, Target>> = RwLock::new(HashMap::new());
    static ref TARGET_NATIVE: RwLock<Target> = RwLock::new(Target::default());
    static ref TARGET_CACHE_UNIQUE_INSERTS: AtomicUsize = AtomicUsize::new(0);
}

fn target_from_string(input: String) -> Result<Target> {
    if input.is_empty() {
        return Ok(Target::default());
    }
    let mut parts = input.split('+');

    let Some(triple_string) = parts.next() else {
        return Err(eyre!("no architecture"));
    };

    let triple = match Triple::from_str(triple_string) {
        Ok(val) => val,
        Err(e) => return Err(eyre!(e)),
    };

    let mut features = CpuFeature::set();
    for flag in parts {
        features.insert(CpuFeature::from_str(flag)?);
    }
    if features.contains(CpuFeature::AVX2) {
        features.insert(CpuFeature::AVX);
    }
    if features.contains(CpuFeature::AVX) {
        features.insert(CpuFeature::SSE42);
    }
    if features.contains(CpuFeature::SSE42) {
        features.insert(CpuFeature::SSE41);
    }
    if features.contains(CpuFeature::SSE41) {
        features.insert(CpuFeature::SSSE3);
    }
    if features.contains(CpuFeature::SSSE3) {
        features.insert(CpuFeature::SSE3);
    }
    if features.contains(CpuFeature::SSE3) {
        features.insert(CpuFeature::SSE2);
    }
    Ok(Target::new(triple, features))
}

/// Populates `TARGET_CACHE` inserting target specified by `description` under `name` key.
/// Additionally, if `native` is set it sets `TARGET_NATIVE` to the specified target.
pub fn target_cache_set(name: String, description: String, native: bool) -> Result<()> {
    let desc_for_log = description.clone();
    let target = target_from_string(description)?;

    if native {
        if !target.is_native() {
            return Err(eyre!("arch not native"));
        }
        let flags_not_supported = Target::default()
            .cpu_features()
            .complement()
            .intersection(*target.cpu_features());
        if !flags_not_supported.is_empty() {
            let mut err_message = String::new();
            err_message.push_str("cpu flags not supported on local cpu for: ");
            for item in flags_not_supported.iter() {
                err_message.push('+');
                err_message.push_str(&item.to_string());
            }
            return Err(eyre!(err_message));
        }
        *TARGET_NATIVE.write() = target.clone();
    }

    let mut cache = TARGET_CACHE.write();
    let is_new_entry = !cache.contains_key(&name);
    cache.insert(name.clone(), target);

    let total = if is_new_entry {
        TARGET_CACHE_UNIQUE_INSERTS.fetch_add(1, Ordering::Relaxed) + 1
    } else {
        TARGET_CACHE_UNIQUE_INSERTS.load(Ordering::Relaxed)
    };
    eprintln!(
        "stylus target cache {}: name=\"{}\" description=\"{}\" unique_entries={}",
        if is_new_entry { "insert" } else { "update" },
        name,
        desc_for_log,
        total
    );

    Ok(())
}

pub fn target_native() -> Target {
    TARGET_NATIVE.read().clone()
}

pub fn target_cache_get(name: &str) -> Result<Target> {
    if name.is_empty() {
        return Ok(TARGET_NATIVE.read().clone());
    }
    TARGET_CACHE
        .read()
        .get(name)
        .cloned()
        .ok_or_eyre("arch not set")
}

pub fn target_cache_unique_inserts() -> usize {
    TARGET_CACHE_UNIQUE_INSERTS.load(Ordering::Relaxed)
}
