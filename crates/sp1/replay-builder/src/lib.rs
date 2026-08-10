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

#[cfg(test)]
mod tests {
    use wasm_encoder::{Module, NameMap, NameSection};

    use super::*;

    /// Builds a minimal wasm module whose custom `name` section names the
    /// given function indices (which must be ascending).
    fn wasm_with_function_names(names: &[(u32, &str)]) -> Vec<u8> {
        let mut name_map = NameMap::new();
        for (index, name) in names {
            name_map.append(*index, name);
        }
        let mut section = NameSection::new();
        section.functions(&name_map);
        let mut module = Module::new();
        module.section(&section);
        module.finish()
    }

    #[test]
    fn extracts_dense_names() {
        let wasm = wasm_with_function_names(&[(0, "a"), (1, "b"), (2, "c")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(
            names,
            [
                Some("a".to_owned()),
                Some("b".to_owned()),
                Some("c".to_owned())
            ]
        );
    }

    #[test]
    fn rebases_indices_and_keeps_gaps() {
        // Named functions start at index 5 (e.g. after unnamed imports) with
        // a hole at 6: the mapping is rebased so the first named function
        // sits at 0, and the hole stays None.
        let wasm = wasm_with_function_names(&[(5, "first"), (7, "third")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(
            names,
            [Some("first".to_owned()), None, Some("third".to_owned())]
        );
    }

    #[test]
    fn errors_without_name_section() {
        let wasm = Module::new().finish();
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(err.to_string().contains("no function names"), "{err:#}");
    }

    #[test]
    fn errors_on_malformed_wasm() {
        let err = extract_function_names(b"not a wasm module").unwrap_err();
        assert!(err.to_string().contains("parse wasm payload"), "{err:#}");
    }

    /// The guest deserializes the mapping from JSON; unnamed slots must
    /// round-trip as `null`, names as plain strings.
    #[test]
    fn json_wire_shape() {
        let wasm = wasm_with_function_names(&[(3, "foo"), (5, "bar")]);
        let names = extract_function_names(&wasm).unwrap();
        let json = serde_json::to_string(&names).unwrap();
        assert_eq!(json, r#"["foo",null,"bar"]"#);
    }
}
