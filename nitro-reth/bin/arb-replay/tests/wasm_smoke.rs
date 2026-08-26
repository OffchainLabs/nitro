//! Executes `arb-replay.wasm` the way a real runner (e.g. the JIT) does: the `arbcompress` imports
//! are bound to native brotli and WASI is stubbed via `caller-env`, proving the module's import ABI
//! end to end.

#![cfg(not(target_family = "wasm"))]

use std::{env, path::PathBuf, process::Command};

use caller_env::{
    ExecEnv, brotli::host as arbcompress, wasip1_stub::host as wasi, wasmer_traits::HasMemory,
};
use wasmer::{
    Function, FunctionEnv, FunctionEnvMut, Instance, Memory, Module, RuntimeError, Store, imports,
};

#[derive(Default)]
struct SmokeEnv {
    memory: Option<Memory>,
    time: u64,
    rand_state: u32,
    stdout: Vec<u8>,
}

impl HasMemory for SmokeEnv {
    fn memory(&self) -> Memory {
        self.memory.clone().expect("memory not set in SmokeEnv")
    }
}

impl ExecEnv for SmokeEnv {
    fn advance_time(&mut self, ns: u64) {
        self.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.rand_state = self.rand_state.wrapping_add(1);
        self.rand_state
    }

    fn print_string(&mut self, bytes: &[u8]) {
        self.stdout.extend_from_slice(bytes);
    }
}

/// Builds `arb-replay.wasm` and returns its path.
fn build_wasm() -> PathBuf {
    let manifest_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let workspace_root = manifest_dir.parent().unwrap().parent().unwrap();
    let status = Command::new(env::var("CARGO").unwrap_or_else(|_| "cargo".into()))
        .args(["build", "-p", "arb-replay", "--target", "wasm32-wasip1"])
        .current_dir(workspace_root)
        .status()
        .expect("failed to spawn cargo");
    assert!(status.success(), "wasm build failed");

    let target_dir = env::var_os("CARGO_TARGET_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| workspace_root.join("target"));
    target_dir.join("wasm32-wasip1/debug/arb-replay.wasm")
}

#[test]
#[ignore = "needs the wasm32-wasip1 target; run by the wasm-replay CI job"]
fn replay_wasm_runs_under_a_jit_style_runner() {
    let wasm = std::fs::read(build_wasm()).expect("read arb-replay.wasm");

    let mut store = Store::default();
    let module = Module::new(&store, wasm).expect("compile module");
    let func_env = FunctionEnv::new(&mut store, SmokeEnv::default());

    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(&mut store, &func_env, $func)
        };
    }
    let imports = imports! {
        "arbcompress" => {
            "brotli_compress" => func!(arbcompress::brotli_compress::<SmokeEnv>),
            "brotli_decompress" => func!(arbcompress::brotli_decompress::<SmokeEnv>),
        },
        "wasi_snapshot_preview1" => {
            "proc_exit" => func!(|_: FunctionEnvMut<SmokeEnv>, code: u32| {
                Err::<(), _>(RuntimeError::new(format!("proc_exit({code})")))
            }),
            "environ_sizes_get" => func!(wasi::environ_sizes_get::<SmokeEnv>),
            "fd_write" => func!(wasi::fd_write::<SmokeEnv>),
            "environ_get" => func!(wasi::environ_get::<SmokeEnv>),
            "fd_close" => func!(wasi::fd_close::<SmokeEnv>),
            "fd_read" => func!(wasi::fd_read::<SmokeEnv>),
            "fd_readdir" => func!(wasi::fd_readdir::<SmokeEnv>),
            "fd_sync" => func!(wasi::fd_sync::<SmokeEnv>),
            "fd_seek" => func!(wasi::fd_seek::<SmokeEnv>),
            "fd_datasync" => func!(wasi::fd_datasync::<SmokeEnv>),
            "path_open" => func!(wasi::path_open::<SmokeEnv>),
            "path_create_directory" => func!(wasi::path_create_directory::<SmokeEnv>),
            "path_remove_directory" => func!(wasi::path_remove_directory::<SmokeEnv>),
            "path_readlink" => func!(wasi::path_readlink::<SmokeEnv>),
            "path_rename" => func!(wasi::path_rename::<SmokeEnv>),
            "path_filestat_get" => func!(wasi::path_filestat_get::<SmokeEnv>),
            "path_unlink_file" => func!(wasi::path_unlink_file::<SmokeEnv>),
            "fd_prestat_get" => func!(wasi::fd_prestat_get::<SmokeEnv>),
            "fd_prestat_dir_name" => func!(wasi::fd_prestat_dir_name::<SmokeEnv>),
            "fd_filestat_get" => func!(wasi::fd_filestat_get::<SmokeEnv>),
            "fd_filestat_set_size" => func!(wasi::fd_filestat_set_size::<SmokeEnv>),
            "fd_pread" => func!(wasi::fd_pread::<SmokeEnv>),
            "fd_pwrite" => func!(wasi::fd_pwrite::<SmokeEnv>),
            "sock_accept" => func!(wasi::sock_accept::<SmokeEnv>),
            "sock_shutdown" => func!(wasi::sock_shutdown::<SmokeEnv>),
            "sched_yield" => func!(wasi::sched_yield::<SmokeEnv>),
            "clock_time_get" => func!(wasi::clock_time_get::<SmokeEnv>),
            "random_get" => func!(wasi::random_get::<SmokeEnv>),
            "args_sizes_get" => func!(wasi::args_sizes_get::<SmokeEnv>),
            "args_get" => func!(wasi::args_get::<SmokeEnv>),
            "poll_oneoff" => func!(wasi::poll_oneoff::<SmokeEnv>),
            "fd_fdstat_get" => func!(wasi::fd_fdstat_get::<SmokeEnv>),
            "fd_fdstat_set_flags" => func!(wasi::fd_fdstat_set_flags::<SmokeEnv>),
        },
    };

    let instance = Instance::new(&mut store, &module, &imports).expect("instantiate");
    let memory = instance
        .exports
        .get_memory("memory")
        .expect("exported memory");
    func_env.as_mut(&mut store).memory = Some(memory.clone());

    let start = instance
        .exports
        .get_typed_function::<(), ()>(&store, "_start")
        .expect("_start export");
    start.call(&mut store).expect("wasm execution");

    let stdout = String::from_utf8(func_env.as_ref(&store).stdout.clone()).expect("utf8 stdout");
    assert!(
        stdout.contains("arb-replay brotli smoke: ok"),
        "unexpected wasm output: {stdout:?}"
    );
}
