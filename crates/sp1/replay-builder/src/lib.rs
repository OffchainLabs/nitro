// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::collections::HashMap;

use anyhow::{Context, ensure, Result};
use wasmparser::{BinaryReader, Name, NameSectionReader, Parser, Payload, TypeRef};

/// Sanity bound on the mapping size; a larger span means a corrupt name
/// section (real replay.wasm has ~15k functions).
const MAX_FUNCTIONS: usize = 1_000_000;

/// Extracts function names from the wasm custom `name` section, indexed by
/// wasmer's `LocalFunctionIndex` (imports dropped, indices shifted down by
/// the import count, unnamed functions `None`).
///
/// Errors on malformed wasm and when no local function has a name.
pub fn extract_function_names(wasm: &[u8]) -> Result<Vec<Option<String>>> {
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

    // Function index space -> local function index space.
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

    /// Minimal module with `num_imports` imported functions and the given
    /// (ascending) function-space names.
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
        let wasm = wasm_module(0, &[(1, "b"), (3, "d")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[None, Some("b"), None, Some("d")]));
    }

    #[test]
    fn shifts_out_imports() {
        let wasm = wasm_module(2, &[(2, "first_local"), (3, "second_local")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[Some("first_local"), Some("second_local")]));
    }

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

    #[test]
    fn errors_on_degenerate_index_span() {
        let wasm = wasm_module(0, &[(0, "a"), (u32::MAX - 1, "corrupt")]);
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(err.to_string().contains("corrupt name section"), "{err:#}");
    }
}
