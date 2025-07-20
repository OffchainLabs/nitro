// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::env;
use std::path::PathBuf;

fn main() {
    let target_arch = env::var("TARGET").unwrap();
    let target_os = env::var("CARGO_CFG_TARGET_OS").unwrap();

    if target_arch.contains("wasm32") {
        println!("cargo:rustc-link-search=../../target/lib-wasm/");
    } else {
        // Logic below fixes the issue when 'cargo build' is run outside of Nitro directory.
        let manifest_dir = env::var("CARGO_MANIFEST_DIR").unwrap();
        let manifest_path = PathBuf::from(manifest_dir);

        // Navigate to the nitro/arbitrator/target/lib directory
        let arbitrator_target_dir = manifest_path
            .parent() // Go up from the crate directory
            .unwrap()
            .join("target")
            .join("lib");

        // Navigate to the nitro/target/lib directory
        let nitro_target_dir = manifest_path
            .parent()
            .unwrap()
            .parent()
            .unwrap()
            .join("target")
            .join("lib");

        println!("cargo:rustc-link-search={}", arbitrator_target_dir.display());
        println!("cargo:rustc-link-search={}", nitro_target_dir.display());

        // Keep the existing relative paths as fallback
        println!("cargo:rustc-link-search=../target/lib/");
        println!("cargo:rustc-link-search=../../target/lib/");
    }

    if target_os == "windows" {
        // Use names without the "-static" suffix for Windows (vcpkg)
        println!("cargo:rustc-link-lib=static=brotlienc");
        println!("cargo:rustc-link-lib=static=brotlidec");
        println!("cargo:rustc-link-lib=static=brotlicommon");
    } else {
        // Use original names for all other platforms (Linux, macOS, etc.)
        println!("cargo:rustc-link-lib=static=brotlienc-static");
        println!("cargo:rustc-link-lib=static=brotlidec-static");
        println!("cargo:rustc-link-lib=static=brotlicommon-static");
    }
}
