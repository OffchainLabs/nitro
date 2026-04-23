use std::{env, fs, path::PathBuf};

const INTERFACES: &str = "nitro-precompile-interfaces";
const SHARED_TYPES: &str = "ArbMultiGasConstraintsTypes.sol";
const GEN_DIR: &str = ".gen";

const FILES: &[(&str, &str)] = &[
    ("nitro-precompile-interfaces/ArbSys.sol", "ArbSys.sol"),
    ("nitro-precompile-interfaces/ArbInfo.sol", "ArbInfo.sol"),
    (
        "nitro-precompile-interfaces/ArbStatistics.sol",
        "ArbStatistics.sol",
    ),
    ("nitro-precompile-interfaces/ArbosTest.sol", "ArbosTest.sol"),
    (
        "nitro-precompile-interfaces/ArbFunctionTable.sol",
        "ArbFunctionTable.sol",
    ),
    (
        "nitro-precompile-interfaces/ArbFilteredTransactionsManager.sol",
        "ArbFilteredTransactionsManager.sol",
    ),
    (
        "nitro-precompile-interfaces/ArbNativeTokenManager.sol",
        "ArbNativeTokenManager.sol",
    ),
    (
        "nitro-precompile-interfaces/ArbWasmCache.sol",
        "ArbWasmCache.sol",
    ),
    ("nitro-precompile-interfaces/ArbDebug.sol", "ArbDebug.sol"),
    (
        "nitro-precompile-interfaces/ArbAddressTable.sol",
        "ArbAddressTable.sol",
    ),
    (
        "nitro-precompile-interfaces/ArbAggregator.sol",
        "ArbAggregator.sol",
    ),
    (
        "nitro-precompile-interfaces/ArbRetryableTx.sol",
        "ArbRetryableTx.sol",
    ),
    ("nitro-precompile-interfaces/ArbWasm.sol", "ArbWasm.sol"),
    (
        "nitro-precompile-interfaces/ArbOwnerPublic.sol",
        "ArbOwnerPublic.sol",
    ),
    (
        "nitro-contracts/src/node-interface/NodeInterface.sol",
        "NodeInterface.sol",
    ),
    (
        "nitro-contracts/src/node-interface/NodeInterfaceDebug.sol",
        "NodeInterfaceDebug.sol",
    ),
];

// The sol! macro does not resolve Solidity `import` statements; inline the
// shared library types into the two files that reference them.
const IMPORTERS: &[&str] = &["ArbGasInfo.sol", "ArbOwner.sol"];

fn main() {
    let manifest = PathBuf::from(env::var_os("CARGO_MANIFEST_DIR").unwrap());
    let gen_root = manifest.join(GEN_DIR);
    fs::create_dir_all(&gen_root).unwrap();

    for (src_rel, dest_name) in FILES {
        let src = manifest.join(src_rel);
        println!("cargo:rerun-if-changed={}", src.display());
        fs::write(gen_root.join(dest_name), strip_natspec(&read(&src))).unwrap();
    }

    let shared = manifest.join(INTERFACES).join(SHARED_TYPES);
    println!("cargo:rerun-if-changed={}", shared.display());
    let shared_body = strip_natspec(&strip_header(&read(&shared)));

    for f in IMPORTERS {
        let src = manifest.join(INTERFACES).join(f);
        println!("cargo:rerun-if-changed={}", src.display());
        let body = strip_natspec(&strip_import(&read(&src), SHARED_TYPES));
        fs::write(gen_root.join(f), format!("{body}\n\n{shared_body}\n")).unwrap();
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

fn strip_natspec(src: &str) -> String {
    let mut out = String::with_capacity(src.len());
    let bytes = src.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        if i + 1 < bytes.len() && bytes[i] == b'/' && bytes[i + 1] == b'*' {
            i += 2;
            while i + 1 < bytes.len() && !(bytes[i] == b'*' && bytes[i + 1] == b'/') {
                i += 1;
            }
            i = (i + 2).min(bytes.len());
            continue;
        }
        if i + 2 < bytes.len() && bytes[i] == b'/' && bytes[i + 1] == b'/' && bytes[i + 2] == b'/' {
            while i < bytes.len() && bytes[i] != b'\n' {
                i += 1;
            }
            continue;
        }
        out.push(bytes[i] as char);
        i += 1;
    }
    out
}
