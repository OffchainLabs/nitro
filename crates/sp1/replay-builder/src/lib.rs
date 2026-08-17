// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::collections::HashMap;

use anyhow::{Context, Result, ensure};
use wasmparser::{BinaryReader, Name, NameSectionReader, Parser, Payload, TypeRef};

/// Sanity bound on the mapping size; a larger declared function count means a
/// corrupt module (real replay.wasm has ~15k functions).
const MAX_FUNCTIONS: usize = 1_000_000;

/// Extracts function names from the wasm custom `name` section, indexed by
/// wasmer's `LocalFunctionIndex`: one entry per defined function (unnamed
/// ones `None`), import names dropped, indices shifted down by the import
/// count.
///
/// Errors on malformed wasm and when no local function has a name.
pub fn extract_function_names(wasm: &[u8]) -> Result<Vec<Option<String>>> {
    let mut name_mapping = HashMap::new();
    let mut num_func_imports: u32 = 0;
    let mut num_local_funcs: u32 = 0;
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
            Payload::FunctionSection(functions) => {
                num_local_funcs = functions.count();
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

    let len = num_local_funcs as usize;
    ensure!(
        len <= MAX_FUNCTIONS,
        "declared function count {len} exceeds {MAX_FUNCTIONS} — corrupt module?"
    );

    // Function index space -> local function index space.
    let mut names = vec![None; len];
    let mut found_local_name = false;
    for (index, name) in name_mapping {
        if index < num_func_imports {
            continue;
        }
        let local = (index - num_func_imports) as usize;
        ensure!(
            local < len,
            "function name index {index} is beyond the declared function count"
        );
        names[local] = Some(name);
        found_local_name = true;
    }
    ensure!(
        found_local_name,
        "no local function names found in the wasm name section"
    );
    Ok(names)
}

#[cfg(test)]
mod tests {
    use wasm_encoder::{
        CodeSection, CustomSection, EntityType, Function, FunctionSection, ImportSection,
        Instruction, Module, NameMap, NameSection, TypeSection,
    };

    use super::*;

    /// Minimal module with `num_imports` imported and `num_locals` defined
    /// functions, naming the given (ascending) function-space indices.
    fn wasm_module(num_imports: u32, num_locals: u32, names: &[(u32, &str)]) -> Vec<u8> {
        let mut module = Module::new();

        if num_imports > 0 || num_locals > 0 {
            let mut types = TypeSection::new();
            types.ty().function([], []);
            module.section(&types);
        }
        if num_imports > 0 {
            let mut imports = ImportSection::new();
            for i in 0..num_imports {
                imports.import("env", &format!("import_{i}"), EntityType::Function(0));
            }
            module.section(&imports);
        }
        if num_locals > 0 {
            let mut functions = FunctionSection::new();
            let mut code = CodeSection::new();
            for _ in 0..num_locals {
                functions.function(0);
                let mut body = Function::new([]);
                body.instruction(&Instruction::End);
                code.function(&body);
            }
            module.section(&functions);
            module.section(&code);
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
        let wasm = wasm_module(0, 3, &[(0, "a"), (1, "b"), (2, "c")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[Some("a"), Some("b"), Some("c")]));
    }

    #[test]
    fn keeps_gaps_for_unnamed_locals() {
        let wasm = wasm_module(0, 4, &[(1, "b"), (3, "d")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[None, Some("b"), None, Some("d")]));
    }

    #[test]
    fn sizes_by_function_count_not_last_name() {
        let wasm = wasm_module(0, 5, &[(1, "b")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(names, owned(&[None, Some("b"), None, None, None]));
    }

    #[test]
    fn shifts_out_imports() {
        let wasm = wasm_module(2, 3, &[(3, "first_named"), (4, "second_named")]);
        let names = extract_function_names(&wasm).unwrap();
        assert_eq!(
            names,
            owned(&[None, Some("first_named"), Some("second_named")])
        );
    }

    #[test]
    fn drops_import_names() {
        let wasm = wasm_module(2, 1, &[(0, "imported"), (2, "local")]);
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
        let wasm = wasm_module(2, 1, &[(0, "imported")]);
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
    fn errors_on_name_index_beyond_function_count() {
        let wasm = wasm_module(0, 2, &[(0, "a"), (u32::MAX - 1, "corrupt")]);
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(
            err.to_string()
                .contains("beyond the declared function count"),
            "{err:#}"
        );
    }

    #[test]
    fn errors_on_excessive_function_count() {
        let wasm = wasm_module(0, MAX_FUNCTIONS as u32 + 1, &[(0, "a")]);
        let err = extract_function_names(&wasm).unwrap_err();
        assert!(err.to_string().contains("exceeds"), "{err:#}");
    }

    /// Pins the JSON the guest deserializes: names as strings, gaps as null.
    #[test]
    fn json_wire_shape() {
        let wasm = wasm_module(0, 3, &[(0, "foo"), (2, "bar")]);
        let names = extract_function_names(&wasm).unwrap();
        let json = serde_json::to_string(&names).unwrap();
        assert_eq!(json, r#"["foo",null,"bar"]"#);
    }
}
