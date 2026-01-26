use anyhow::Result;
use arbutil::Bytes32;
use std::collections::HashSet;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::str::FromStr;
use tracing::debug;
use tracing::warn;

use crate::config::ModuleRootConfig;

#[derive(Debug, Clone)]
pub struct MachineLocator {
    root_path: PathBuf,
    latest: Bytes32,
    module_roots: Vec<Bytes32>,
}

impl MachineLocator {
    pub fn new(module_root_config: &ModuleRootConfig) -> Result<Self> {
        let mut dirs = Vec::new();
        let mut module_roots_set = HashSet::new();
        let mut latest_module_root = Bytes32::default();
        let mut final_root_path = PathBuf::new();

        // Use CARGO_MANIFEST_DIR to find the crate root.
        let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));

        if let Some(module_root) = module_root_config.module_root {
            module_roots_set.insert(module_root);
            latest_module_root = module_root;

            final_root_path =
                if let Some(grandparent) = manifest_dir.parent().and_then(|p| p.parent()) {
                    grandparent.join("target").join("machines")
                } else {
                    final_root_path
                };
        } else {
            if let Some(rp) = module_root_config
                .module_root_path
                .as_ref()
                .filter(|s| s.exists())
            {
                dirs.push(PathBuf::from(rp));
            } else {
                // Try to find the workspace root by looking for "target" in common locations
                // <crate_root>/../../target/machines
                if let Some(grandparent) = manifest_dir.parent().and_then(|p| p.parent()) {
                    dirs.push(grandparent.join("target").join("machines"));
                }
                // <crate_root>/target/machines
                dirs.push(manifest_dir.join("target").join("machines"));

                // Check working directory
                if let Ok(work_dir) = env::current_dir() {
                    dirs.push(work_dir.join("machines"));
                    dirs.push(work_dir.join("target").join("machines"));
                }

                // Check relative to executable
                if let Ok(exec_path) = env::current_exe() {
                    if let Some(grandparent_of_exec) = exec_path.parent().and_then(|p| p.parent()) {
                        dirs.push(grandparent_of_exec.join("machines"));
                    }
                }
            }
        }

        for dir in dirs {
            if !dir.exists() || !dir.is_dir() {
                debug!("{dir:?} does not exist!!!");
                continue;
            }

            let entries = match fs::read_dir(&dir) {
                Ok(e) => e,
                Err(e) => {
                    warn!("Reading directory {:?} error: {}", dir, e);
                    continue;
                }
            };

            for entry in entries.flatten() {
                let mr_file = entry.path().join("module-root.txt");

                if !mr_file.exists() {
                    continue;
                }

                let mr_content = match fs::read_to_string(&mr_file) {
                    Ok(c) => c,
                    Err(e) => {
                        warn!("Reading module roots file {:?} error: {}", mr_file, e);
                        continue;
                    }
                };

                let module_root = match Bytes32::from_str(mr_content.trim()) {
                    Ok(h) => h,
                    Err(_) => {
                        warn!("Error converting module root file {mr_file:?} into hash");
                        continue;
                    }
                };

                let dir_name = entry.file_name().to_string_lossy().to_string();

                // IMPORTANT:
                // Go's moduleRoot.Hex() returns "0x" + hex.
                // Rust Bytes32 Display impl returns raw hex.
                // We must format it manually to match Go's directory naming convention.
                let module_root_hex = format!("0x{}", module_root);

                if dir_name != "latest" && dir_name != module_root_hex {
                    continue;
                }

                module_roots_set.insert(module_root);

                if dir_name == "latest" {
                    latest_module_root = module_root;
                }

                final_root_path = dir.canonicalize().unwrap_or(dir.clone());
            }

            if !final_root_path.as_os_str().is_empty() {
                break;
            }
        }

        let module_roots: Vec<Bytes32> = module_roots_set.into_iter().collect();

        Ok(MachineLocator {
            root_path: final_root_path,
            latest: latest_module_root,
            module_roots,
        })
    }

    pub fn get_machine_path(&self, module_root: Bytes32) -> PathBuf {
        if module_root == Bytes32::default() || module_root == self.latest {
            self.root_path.join("latest")
        } else {
            self.root_path.join(format!("0x{}", module_root))
        }
    }

    pub fn latest_wasm_module_root(&self) -> Bytes32 {
        self.latest
    }

    pub fn module_roots(&self) -> &[Bytes32] {
        &self.module_roots
    }
}

#[cfg(test)]
mod tests {
    use std::str::FromStr;

    use arbutil::Bytes32;

    use crate::{config::ModuleRootConfig, server_jit::machine_locator::MachineLocator};

    #[test]
    fn test_new_machine_locator_with_path() {
        let locator_path = "testdata".to_owned();
        let config = ModuleRootConfig {
            module_root: None,
            module_root_path: Some(locator_path.into()),
        };
        let machine_locator = MachineLocator::new(&config).unwrap();

        let expected_latest =
            Bytes32::from_str("0xf4389b835497a910d7ba3ebfb77aa93da985634f3c052de1290360635be40c4a")
                .unwrap();
        assert_eq!(expected_latest, machine_locator.latest);

        let expected_module_roots = [
            Bytes32::from_str("0x68e4fe5023f792d4ef584796c84d710303a5e12ea02d6e37e2b5e9c4332507c4")
                .unwrap(),
            Bytes32::from_str("0x8b104a2e80ac6165dc58b9048de12f301d70b02a0ab51396c22b4b4b802a16a4")
                .unwrap(),
            Bytes32::from_str("0xf4389b835497a910d7ba3ebfb77aa93da985634f3c052de1290360635be40c4a")
                .unwrap(),
        ];

        let mut module_roots = machine_locator.module_roots().to_vec();
        module_roots.sort();
        let module_roots: [Bytes32; 3] = module_roots.try_into().unwrap();

        assert_eq!(expected_module_roots, module_roots);
    }

    #[test]
    fn test_new_machine_locator_with_module_root() {
        let config = ModuleRootConfig {
            module_root: Some(
                Bytes32::from_str(
                    "0x68e4fe5023f792d4ef584796c84d710303a5e12ea02d6e37e2b5e9c4332507c4",
                )
                .unwrap(),
            ),
            module_root_path: None,
        };
        let machine_locator = MachineLocator::new(&config).unwrap();

        let expected_latest =
            Bytes32::from_str("0x68e4fe5023f792d4ef584796c84d710303a5e12ea02d6e37e2b5e9c4332507c4")
                .unwrap();
        assert_eq!(expected_latest, machine_locator.latest);

        let expected_module_roots = [Bytes32::from_str(
            "0x68e4fe5023f792d4ef584796c84d710303a5e12ea02d6e37e2b5e9c4332507c4",
        )
        .unwrap()];

        let mut module_roots = machine_locator.module_roots().to_vec();
        module_roots.sort();
        let module_roots: [Bytes32; 1] = module_roots.try_into().unwrap();

        assert_eq!(expected_module_roots, module_roots);
    }
}
