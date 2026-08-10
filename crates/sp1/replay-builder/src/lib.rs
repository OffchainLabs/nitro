// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::collections::HashMap;

use anyhow::Context;
use wasmparser::{BinaryReader, Name, NameSectionReader, Parser, Payload};

/// Extracts the original function names from the wasm module's custom `name`
/// section.
///
/// Returns one entry per function index (functions without a name are
/// `None`), with indices rebased so that the first named function sits at 0.
/// The replay guest consumes this mapping (serialized as JSON) to register
/// profiler symbols for wasmer-compiled code.
///
/// Errors on malformed wasm and on modules without any function names: the
/// only expected input is replay.wasm, which is always built with a name
/// section, so a missing one indicates a broken build.
pub fn extract_function_names(wasm: &[u8]) -> anyhow::Result<Vec<Option<String>>> {
    let mut name_mapping = HashMap::new();
    for payload in Parser::new(0).parse_all(wasm) {
        let payload = payload.context("parse wasm payload")?;
        let Payload::CustomSection(section) = payload else {
            continue;
        };
        if section.name() != "name" {
            continue;
        }
        let reader =
            NameSectionReader::new(BinaryReader::new(section.data(), section.data_offset()));
        for name in reader {
            let Name::Function(name_map) = name.context("parse name subsection")? else {
                continue;
            };
            for naming in name_map {
                let naming = naming.context("parse function naming")?;
                name_mapping.insert(naming.index, naming.name.to_string());
            }
        }
    }

    // Names can be sparse; rebase indices to the first named function and
    // leave the gaps as `None`.
    let min_index = name_mapping
        .keys()
        .copied()
        .min()
        .context("no function names found in the wasm name section")?;
    let max_index = *name_mapping.keys().max().unwrap(); // non-empty: min() succeeded
    let mut names = vec![None; (max_index - min_index) as usize + 1];
    for (index, name) in name_mapping {
        names[(index - min_index) as usize] = Some(name);
    }
    Ok(names)
}
