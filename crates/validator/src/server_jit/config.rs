use std::{
    env,
    path::{Path, PathBuf},
};

use anyhow::{anyhow, Context, Result};

#[derive(Clone, Debug)]
pub struct JitMachineConfig {
    pub prover_bin_path: String,
    pub jit_cranelift: bool,
    pub wasm_memory_usage_limit: u64,
    pub jit_path: Option<String>,
}

impl Default for JitMachineConfig {
    fn default() -> Self {
        Self {
            prover_bin_path: "replay.wasm".to_owned(),
            jit_cranelift: true,
            wasm_memory_usage_limit: 4_294_967_296,
            jit_path: None,
        }
    }
}

pub fn get_jit_path(config_path: &Option<String>) -> Result<PathBuf> {
    // 1. If a custom path is provided, use it directly
    if let Some(jit_path) = config_path {
        let path = Path::new(&jit_path);
        if path.exists() {
            return Ok(path.to_path_buf());
        }
        return Err(anyhow!(
            "Custom JIT path provided but not found: {jit_path}",
        ));
    }

    // 2. Fall back to auto-detection
    let current_exe = env::current_exe().context("failed to get path of current executable")?;

    let exe_string = current_exe.to_string_lossy();
    let is_test_env = exe_string.contains("test")
        || exe_string.contains("deps")
        || exe_string.contains("system_tests");

    let candidate = if is_test_env {
        // CARGO_MANIFEST_DIR points to crates/validator, therefore we need to look for the grandparent
        let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
        if let Some(grandparent) = manifest_dir.parent().and_then(|p| p.parent()) {
            grandparent.join("target").join("bin").join("jit")
        } else {
            return Err(anyhow!(
                "Custom JIT path not found for test env: {manifest_dir:?}",
            ));
        }
    } else {
        current_exe
            .parent()
            .ok_or_else(|| anyhow!("failed to resolve parent directory of executable"))?
            .join("jit")
    };

    if candidate.exists() {
        return Ok(candidate);
    }

    // 3. Fallback: Search system PATH
    // We treat a missing PATH var as "just continue" rather than a hard error
    if let Ok(path_var) = env::var("PATH") {
        for split_path in env::split_paths(&path_var) {
            let joined = split_path.join("jit");
            if joined.exists() {
                return Ok(joined);
            }
        }
    }

    Err(anyhow!(
        "jit binary not found in local paths or system PATH"
    ))
}

#[cfg(test)]
mod tests {
    use crate::server_jit::config::get_jit_path;

    #[test]
    fn test_get_jit_path() {
        let jit_path = get_jit_path(&None).unwrap();

        assert!(jit_path.exists(), "JIT binary does not exist");
        assert!(
            jit_path.is_file(),
            "JIT path points to a directory, expected a file"
        );

        let path_str = jit_path.to_str().expect("path contains invalid utf-8");

        assert!(
            path_str.contains("nitro/target/bin/jit"),
            "Path {:?} did not contain expected substring 'nitro/target/bin/jit'",
            jit_path
        );
    }
}
