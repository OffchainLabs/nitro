// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use eyre::{eyre, OptionExt, Result};
use lazy_static::lazy_static;
use parking_lot::RwLock;
use std::{collections::HashMap, str::FromStr};
use wasmer_types::{CpuFeature, Target, Triple};

use crate::cache::InitCache;

lazy_static! {
    static ref TARGET_CACHE: RwLock<HashMap<String, Target>> = RwLock::new(HashMap::new());
}

fn target_from_string(input: String) -> Result<Target> {
    let mut parts = input.split('+');

    let Some(trip_sting) = parts.next() else {
        return Err(eyre!("no architecture"));
    };

    let trip = match Triple::from_str(trip_sting) {
        Ok(val) => val,
        Err(e) => return Err(eyre!(e)),
    };

    let mut features = CpuFeature::set();
    for flag in parts {
        features.insert(CpuFeature::from_str(flag)?);
    }

    Ok(Target::new(trip, features))
}

pub fn target_cache_set(name: String, description: String, native: bool) -> Result<()> {
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
        InitCache::set_target(target.clone())
    }

    TARGET_CACHE.write().insert(name, target);

    Ok(())
}

pub fn target_cache_get(name: &str) -> Result<Target> {
    if name.is_empty() {
        return Ok(InitCache::target());
    }
    TARGET_CACHE
        .read()
        .get(name)
        .cloned()
        .ok_or_eyre("arch not set")
}
