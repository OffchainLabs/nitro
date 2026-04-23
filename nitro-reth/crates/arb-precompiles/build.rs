use std::{env, fs, path::PathBuf};

const SUBMODULE: &str = "nitro-precompile-interfaces";
const SHARED: &str = "ArbMultiGasConstraintsTypes.sol";
const IMPORTERS: &[&str] = &["ArbGasInfo.sol", "ArbOwner.sol"];
const GEN_DIR: &str = ".gen";

fn main() {
    let manifest = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap());
    let src_root = manifest.join(SUBMODULE);
    let gen_root = manifest.join(GEN_DIR);
    fs::create_dir_all(&gen_root).unwrap();

    println!("cargo:rerun-if-changed={}", src_root.join(SHARED).display());
    let shared_body = strip_header(&read(&src_root.join(SHARED)));

    for f in IMPORTERS {
        let src = src_root.join(f);
        println!("cargo:rerun-if-changed={}", src.display());
        let body = strip_import(&read(&src), SHARED);
        let merged = format!("{body}\n\n{shared_body}\n");
        fs::write(gen_root.join(f), merged).unwrap();
    }
}

fn read(p: &PathBuf) -> String {
    fs::read_to_string(p).unwrap_or_else(|e| panic!("read {p:?}: {e}"))
}

fn strip_header(src: &str) -> String {
    src.lines()
        .filter(|l| !l.starts_with("pragma") && !l.starts_with("// SPDX"))
        .collect::<Vec<_>>()
        .join("\n")
}

fn strip_import(src: &str, target: &str) -> String {
    src.lines()
        .filter(|l| !(l.starts_with("import") && l.contains(target)))
        .collect::<Vec<_>>()
        .join("\n")
}
