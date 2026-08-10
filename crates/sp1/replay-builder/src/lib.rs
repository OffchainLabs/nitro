// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::collections::HashMap;

use anyhow::{Context, ensure};
use wasmparser::{BinaryReader, Name, NameSectionReader, Parser, Payload, TypeRef};

/// Upper bound on the size of the emitted mapping. The real replay.wasm has
/// tens of thousands of functions; a span beyond this indicates a corrupt
/// name section, which would otherwise make the builder allocate gigabytes.
const MAX_FUNCTIONS: usize = 10_000_000;

/// Extracts the original function names from the wasm module's custom `name`
/// section.
///
/// The name section indexes the wasm *function index space*, where imported
/// functions occupy the lowest indices; the replay guest looks names up by
/// wasmer's `LocalFunctionIndex`, which counts *defined* functions only. The
/// returned vector is therefore indexed by local function index: import names
/// are dropped and the remaining indices are shifted down by the number of
/// imported functions. Functions without a name are `None`. The guest
/// consumes this mapping (serialized as JSON) to register profiler symbols
/// for wasmer-compiled code.
///
/// Errors on malformed wasm and on modules without any local function names:
/// the only expected input is replay.wasm, which is always built with a name
/// section, so a missing one indicates a broken build.
pub fn extract_function_names(wasm: &[u8]) -> anyhow::Result<Vec<Option<String>>> {
    let mut name_mapping = HashMap::new();
    let mut num_func_imports: u32 = 0;
    for payload in Parser::new(0).parse_all(wasm) {
        let payload = payload.context("parse wasm payload")?;
        match payload {
            Payload::ImportSection(imports) => {
                for import in imports.into_imports() {
                    let import = import.context("parse import")?;
                    if matches!(import.ty, TypeRef::Func(_)) {
                        num_func_imports += 1;
                    }
                }
            }
            Payload::CustomSection(section) if section.name() == "name" => {
                let reader = NameSectionReader::new(BinaryReader::new(
                    section.data(),
                    section.data_offset(),
                ));
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
            _ => {}
        }
    }

    // Translate from the function index space to the local function index
    // space: drop import names, shift the rest down by the import count.
    let max_local = name_mapping
        .keys()
        .filter(|&&index| index >= num_func_imports)
        .map(|&index| (index - num_func_imports) as usize)
        .max()
        .context("no local function names found in the wasm name section")?;

    let len = max_local + 1;
    ensure!(
        len <= MAX_FUNCTIONS,
        "function name span {len} exceeds {MAX_FUNCTIONS} entries — corrupt name section?"
    );

    let mut names = vec![None; len];
    for (index, name) in name_mapping {
        if index >= num_func_imports {
            names[(index - num_func_imports) as usize] = Some(name);
        }
    }
    Ok(names)
}

#[cfg(test)]
mod tests {
    use wasm_encoder::{
        CustomSection, EntityType, ImportSection, Module, NameMap, NameSection, TypeSection,
    };

    use super::*;

    /// Builds a minimal wasm module with `num_imports` imported functions
    /// whose custom `name` section names the given function-space indices
    /// (which must be ascending).
    fn wasm_module(num_imports: u32, names: &[(u32, &str)]) -> Vec<u8> {
        let mut module = Module::new();

        if num_imports > 0 {
            let mut types = TypeSection::new();
            types.ty().function([], []);
            let mut imports = ImportSection::new();
            for i in 0..num_imports {
                imports.import("env", &format!("import_{i}"), EntityType::Function(0));
            }
            module.section(&types);
            module.section(&imports);
        }

        let mut name_map = NameMap::new();
        for (index, name) in names {
            name_map.append(*index, name);
        }
        let mut section = NameSection::new();
        section.functions(&name_map);
        module.section(&section);

        module.finish()
    }

    fn owned(names: &[Option<&str>]) -> Vec<Option<String>> {
        names.iter().map(|n| n.map(str::to_owned)).collect()
    }

    #[test]
    fn extracts_dense_names() {
        let wasm = wasm_module(0, &[(0, "a"), (1, "b"), (2, "c")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[Some("a"), Some("b"), Some("c")]));
    }

    #[test]
    fn keeps_gaps_for_unnamed_locals() {
        // No imports: function-space indices are local indices. Unnamed
        // locals keep their slots so the guest's local-index lookups stay
        // aligned.
        let wasm = wasm_module(0, &[(1, "b"), (3, "d")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[None, Some("b"), None, Some("d")]));
    }

    /// Imported functions occupy the lowest function-space indices; the
    /// mapping is shifted down by the import count so it aligns with
    /// wasmer's local function index space.
    #[test]
    fn shifts_out_imports() {
        let wasm = wasm_module(2, &[(2, "first_local"), (3, "second_local")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[Some("first_local"), Some("second_local")]));
    }

    /// Names attached to imports are dropped: imports are not part of the
    /// local function index space the guest indexes with.
    #[test]
    fn drops_import_names() {
        let wasm = wasm_module(2, &[(0, "imported"), (2, "local")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[Some("local")]));
    }

    #[test]
    fn errors_without_name_section() {
        let wasm = Module::new().finish();
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(
            err.to_string().contains("no local function names"),
            "{err:#}"
        );
    }

    /// A name section that only names imports leaves the local space empty.
    #[test]
    fn errors_with_only_import_names() {
        let wasm = wasm_module(2, &[(0, "imported")]);
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(
            err.to_string().contains("no local function names"),
            "{err:#}"
        );
    }

    #[test]
    fn errors_on_malformed_wasm() {
        let err = extract_function_names(b"not a wasm module").unwrap_err();
        assert!(err.to_string().contains("parse wasm payload"), "{err:#}");
    }

    /// Garbage inside an otherwise valid `name` custom section must error,
    /// not be skipped.
    #[test]
    fn errors_on_malformed_name_section() {
        let mut module = Module::new();
        module.section(&CustomSection {
            name: "name".into(),
            data: b"\xff\xff garbage".as_slice().into(),
        });
        let err = extract_function_names(&module.finish()).unwrap_err();
        assert!(err.to_string().contains("parse name subsection"), "{err:#}");
    }

    /// A corrupt index would otherwise size the output vector into the
    /// gigabytes; the span guard turns that into an error.
    #[test]
    fn errors_on_degenerate_index_span() {
        let wasm = wasm_module(0, &[(0, "a"), (u32::MAX - 1, "corrupt")]);
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(err.to_string().contains("corrupt name section"), "{err:#}");
    }

    /// The guest deserializes the mapping from JSON; unnamed slots must
    /// round-trip as `null`, names as plain strings.
    #[test]
    fn json_wire_shape() {
        let wasm = wasm_module(0, &[(0, "foo"), (2, "bar")]);
        let names = extract_function_names(&wasm).unwrap();
        let json = serde_json::to_string(&names).unwrap();
        assert_eq!(json, r#"["foo",null,"bar"]"#);
    }
}
