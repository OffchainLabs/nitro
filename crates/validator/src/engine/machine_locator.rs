// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use std::collections::HashSet;
use std::env;
use std::fs;
use std::path::{Path, PathBuf};
use std::str::FromStr;
use tracing::debug;
use tracing::info;
use tracing::warn;

use crate::engine::ModuleRoot;

#[derive(Debug, Default, Clone, PartialEq, Eq, Hash)]
pub struct ModuleRootMeta {
    /// Module root
    pub module_root: ModuleRoot,
    /// Directory where `module_root` is stored
    pub path: PathBuf,
}

#[derive(Debug, Clone)]
pub struct MachineLocator {
    root_path: PathBuf,
    latest: ModuleRootMeta,
    module_roots: Vec<ModuleRootMeta>,
}

impl MachineLocator {
    pub fn new(root_path: &Option<PathBuf>) -> Result<Self> {
        let mut dirs = Vec::new();
        let mut module_roots_set = HashSet::new();
        let mut latest_module_root = ModuleRootMeta::default();
        let mut final_root_path = PathBuf::new();

        if let Some(root_path) = root_path {
            dirs.push(root_path.clone());
        } else {
            // Use CARGO_MANIFEST_DIR to find the crate root.
            let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));

            // Try to find the workspace root by looking for "target" in common locations
            // <crate_root>/../../target/machines
            if let Some(grandparent) = manifest_dir.parent().and_then(|p| p.parent()) {
                dirs.push(grandparent.join("target").join("machines"));
            }

            // Check working directory
            if let Ok(work_dir) = env::current_dir() {
                dirs.push(work_dir.join("machines"));
                dirs.push(work_dir.join("target").join("machines"));
            }

            // Check relative to the executable
            if let Ok(exec_path) = env::current_exe() {
                if let Some(grandparent_of_exec) = exec_path.parent().and_then(|p| p.parent()) {
                    dirs.push(grandparent_of_exec.join("machines"));
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
                    warn!("Reading directory {dir:?} error: {e:?}");
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
                        warn!("Reading module roots file {mr_file:?} error: {e:?}");
                        continue;
                    }
                };

                let module_root = match ModuleRoot::from_str(mr_content.trim()) {
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
                let module_root_hex = format!("0x{module_root}");

                if dir_name != "latest" && dir_name != module_root_hex {
                    continue;
                }

                let root_meta = ModuleRootMeta {
                    module_root,
                    path: entry.path(),
                };

                module_roots_set.insert(root_meta);

                if dir_name == "latest" {
                    latest_module_root = ModuleRootMeta {
                        module_root,
                        path: entry.path(),
                    };
                }

                final_root_path = dir.canonicalize().unwrap_or(dir.clone());
            }

            if !final_root_path.as_os_str().is_empty() {
                break;
            }
        }

        // Log found machines. This is okay since it's only called on server startup
        let module_roots: Vec<ModuleRootMeta> = module_roots_set
            .into_iter()
            .inspect(|meta| info!("Machine locator found module root at path: {:?}", meta.path))
            .collect();

        Ok(MachineLocator {
            root_path: final_root_path,
            latest: latest_module_root,
            module_roots,
        })
    }

    pub fn get_machine_path(&self, module_root: ModuleRoot) -> PathBuf {
        if module_root == ModuleRoot::default() || module_root == self.latest.module_root {
            self.root_path.join("latest")
        } else {
            self.root_path.join(format!("0x{module_root}"))
        }
    }

    pub fn latest_wasm_module_root(&self) -> &ModuleRootMeta {
        &self.latest
    }

    pub fn module_roots(&self) -> &[ModuleRootMeta] {
        &self.module_roots
    }
}

#[cfg(test)]
mod tests {
    use std::{
        path::{Path, PathBuf},
        str::FromStr,
    };

    use crate::engine::{
        machine_locator::{MachineLocator, ModuleRootMeta},
        ModuleRoot,
    };
    use anyhow::{anyhow, Result};
    use arbutil::Bytes32;
    use rand::RngCore;

    fn get_or_create_root_path(machines_dir: &PathBuf, root: &str) -> RootMetaWrapper {
        let complete_root_path = machines_dir.join(root);

        let (module_root, should_delete) = if !complete_root_path.exists() {
            // We assume "latest" module root exist. That avoids multiple tests trying to create
            // the same "latest" module root to run into each other; be run in parallel
            if root == "latest" {
                panic!("latest path should exist!")
            };

            std::fs::create_dir_all(&complete_root_path)
                .expect("Failed to create target/machines directory");

            std::fs::write(complete_root_path.join("module-root.txt"), root)
                .expect("Failed to write module-root.txt");
            let module_root = ModuleRoot::from_str(root).unwrap();
            // Since we are the ones who are creating the directory for this module root, we mark it
            // for deletion
            (module_root, true)
        } else {
            let existing_content =
                std::fs::read_to_string(complete_root_path.join("module-root.txt"))
                    .expect("Failed to read existing module-root.txt");
            let module_root = ModuleRoot::from_str(&existing_content.trim()).unwrap();
            // If there's an existing module root we still want to remember it; however, we don't want
            // to delete on drop since we were not the one who created it
            (module_root, false)
        };

        RootMetaWrapper {
            root_meta: ModuleRootMeta {
                module_root,
                path: complete_root_path,
            },
            should_delete,
        }
    }

    fn gen_random_module_root() -> ModuleRoot {
        let mut bytes = [0u8; 32];
        rand::rng().fill_bytes(&mut bytes);

        Bytes32(bytes)
    }

    #[derive(Debug, Clone)]
    struct RootMetaWrapper {
        root_meta: ModuleRootMeta,
        // dictates if root_meta should be deleted on drop. Only module roots
        // that have been created by LocatorSimulator are marked for deletion
        should_delete: bool,
    }

    struct LocatorSimulator {
        root_metas: Vec<RootMetaWrapper>,
        latest_root: RootMetaWrapper,
    }

    impl LocatorSimulator {
        // Generates a new LocatorSimulator by creating temporary module root
        // folders with their respective module-root.txt so that MachineLocator
        // can find them
        fn new(root_count: u32) -> Self {
            let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
            assert!(root_count > 0, "there must be at least one module root");

            let machines_dir = manifest_dir
                .ancestors()
                .nth(2)
                .expect("Failed to navigate to workspace root")
                .join("target/machines");

            let mut root_metas = vec![];

            for _ in 0..root_count {
                let random_module_root = gen_random_module_root();
                let module_root_str = format!("0x{random_module_root}");
                let root_meta = get_or_create_root_path(&machines_dir, &module_root_str);

                root_metas.push(root_meta);
            }

            let latest_root = get_or_create_root_path(&machines_dir, "latest");
            root_metas.push(latest_root.clone());

            LocatorSimulator {
                root_metas,
                latest_root,
            }
        }
    }

    impl Drop for LocatorSimulator {
        fn drop(&mut self) {
            for root_meta_wrapper in self.root_metas.iter() {
                if root_meta_wrapper.should_delete && root_meta_wrapper.root_meta.path.exists() {
                    if let Err(e) = std::fs::remove_dir_all(&root_meta_wrapper.root_meta.path) {
                        eprintln!(
                            "Failed to cleanup test directory {:?}: {}",
                            root_meta_wrapper.root_meta.path, e
                        );
                    }
                }
            }
        }
    }

    fn test_machine_locator(root_count: u32, root_path: &Option<PathBuf>) -> Result<()> {
        let file_manager = LocatorSimulator::new(root_count);
        let machine_locator = MachineLocator::new(root_path)?;

        if machine_locator.module_roots().is_empty() {
            return Err(anyhow!("empty module roots"));
        }

        assert_eq!(
            machine_locator.latest_wasm_module_root().module_root,
            file_manager.latest_root.root_meta.module_root
        );

        for root_meta_wrapper in file_manager.root_metas.iter() {
            assert!(machine_locator
                .module_roots()
                .contains(&root_meta_wrapper.root_meta));
        }

        // Check if get_machine_path returns the correct module root path for
        // the simulated module roots. Only the last module root in root_metas
        // is the "latest" one, all the rest should be "0x..."
        file_manager
            .root_metas
            .iter()
            .take(root_count as usize)
            .for_each(|root_meta_wrapper| {
                // let root_meta_wrapper = file_manager.root_metas.first().unwrap();
                let mod_root = root_meta_wrapper.root_meta.module_root;
                let module_root = machine_locator.get_machine_path(mod_root);
                assert!(module_root
                    .to_str()
                    .unwrap()
                    .contains(&format!("0x{mod_root}")));
            });

        Ok(())
    }

    #[test]
    fn test_machine_locator_one_machine() -> Result<()> {
        test_machine_locator(1, &None)
    }

    #[test]
    fn test_machine_locator_many_machines() -> Result<()> {
        test_machine_locator(10, &None)
    }

    #[test]
    fn test_machine_locator_with_root_path() -> Result<()> {
        let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
        let machines_dir = manifest_dir
            .ancestors()
            .nth(2)
            .expect("Failed to navigate to workspace root")
            .join("target/machines");

        test_machine_locator(2, &Some(machines_dir))?;

        Ok(())
    }

    #[test]
    fn test_machine_locator_wrong_root_path() -> Result<()> {
        let result = test_machine_locator(2, &Some("i/do/not/exist".into()));
        assert!(result.is_err());

        let error = result.err().unwrap();
        let err_str = error.to_string();
        assert!(err_str.contains("empty module roots"));

        Ok(())
    }
}
