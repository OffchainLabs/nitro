//! Executes `arb-replay.wasm` under a JIT-style wasmer runner with *mocked*
//! `arbcompress` imports. The mocks are only consistent with each other (marker
//! framing instead of real compression), so the guest's round-trip succeeds only
//! if both calls crossed the import boundary — silently reaching any real brotli
//! would trap. The mocks also record the guest's arguments, pinning the ABI.

#![cfg(not(target_family = "wasm"))]

use std::{env, path::PathBuf, process::Command};

use caller_env::{
    ExecEnv, GuestPtr, MemAccess,
    wasip1_stub::host as wasi,
    wasmer_traits::{HasMemory, WasmerMem},
};
use nitro_brotli::{BrotliStatus, DEFAULT_WINDOW_SIZE, Dictionary};
use wasmer::{
    Function, FunctionEnv, FunctionEnvMut, Instance, Memory, Module, RuntimeError, Store, imports,
};

/// Prefix the mock compressor frames its output with.
const MOCK_MARKER: &[u8] = b"MOCK";

#[derive(Default)]
struct RunnerEnv {
    memory: Option<Memory>,
    time: u64,
    rand_state: u32,
    stdout: Vec<u8>,
    compress_params: Option<(u32, u32, Dictionary)>,
    decompress_dictionary: Option<Dictionary>,
}

impl HasMemory for RunnerEnv {
    fn memory(&self) -> Memory {
        self.memory.clone().expect("memory not set in RunnerEnv")
    }
}

impl ExecEnv for RunnerEnv {
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

/// Writes `data` to the guest's output buffer, honoring the in/out length pointer.
fn write_output(mem: &mut WasmerMem, out_buf_ptr: GuestPtr, out_len_ptr: GuestPtr, data: &[u8]) {
    let capacity = mem.read_u32(out_len_ptr);
    assert!(
        data.len() as u32 <= capacity,
        "guest output buffer too small: {} > {capacity}",
        data.len()
    );
    mem.write_slice(out_buf_ptr, data);
    mem.write_u32(out_len_ptr, data.len() as u32);
}

/// Mock `arbcompress.brotli_compress`: frames the input as `MOCK + input`.
#[allow(clippy::too_many_arguments)]
fn mock_compress(
    mut ctx: FunctionEnvMut<RunnerEnv>,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> BrotliStatus {
    let (data, store) = ctx.data_and_store_mut();
    data.compress_params = Some((level, window_size, dictionary));
    let mut mem = WasmerMem::new(data.memory(), store);

    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let framed = [MOCK_MARKER, &input].concat();
    write_output(&mut mem, out_buf_ptr, out_len_ptr, &framed);
    BrotliStatus::Success
}

/// Mock `arbcompress.brotli_decompress`: strips the `MOCK` frame.
fn mock_decompress(
    mut ctx: FunctionEnvMut<RunnerEnv>,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: Dictionary,
) -> BrotliStatus {
    let (data, store) = ctx.data_and_store_mut();
    data.decompress_dictionary = Some(dictionary);
    let mut mem = WasmerMem::new(data.memory(), store);

    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let payload = input
        .strip_prefix(MOCK_MARKER)
        .expect("guest passed data the mock compressor did not produce");
    write_output(&mut mem, out_buf_ptr, out_len_ptr, payload);
    BrotliStatus::Success
}

/// Extracts `(in, out)` from the guest's "(<in> -> <out> bytes)" report.
fn parse_sizes(stdout: &str) -> (usize, usize) {
    let (_, tail) = stdout.split_once('(').expect("no sizes in guest output");
    let (in_len, tail) = tail.split_once(" -> ").expect("malformed size report");
    let (out_len, _) = tail.split_once(" bytes").expect("malformed size report");
    (in_len.parse().unwrap(), out_len.parse().unwrap())
}

/// Builds `arb-replay.wasm` in release mode (the profile we ship) and returns its path.
fn build_wasm() -> PathBuf {
    let manifest_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let workspace_root = manifest_dir.parent().unwrap().parent().unwrap();
    let status = Command::new(env::var("CARGO").unwrap_or_else(|_| "cargo".into()))
        .args([
            "build",
            "-p",
            "arb-replay",
            "--release",
            "--target",
            "wasm32-wasip1",
        ])
        .current_dir(workspace_root)
        .status()
        .expect("failed to spawn cargo");
    assert!(status.success(), "wasm build failed");

    let target_dir = env::var_os("CARGO_TARGET_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| workspace_root.join("target"));
    target_dir.join("wasm32-wasip1/release/arb-replay.wasm")
}

#[test]
#[ignore = "needs the wasm32-wasip1 target; run by the wasm-replay CI job"]
fn replay_wasm_routes_brotli_through_the_runner_imports() {
    let wasm = std::fs::read(build_wasm()).expect("read arb-replay.wasm");

    let mut store = Store::default();
    let module = Module::new(&store, wasm).expect("compile module");
    let func_env = FunctionEnv::new(&mut store, RunnerEnv::default());

    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(&mut store, &func_env, $func)
        };
    }
    let mut imports = imports! {
        "arbcompress" => {
            "brotli_compress" => func!(mock_compress),
            "brotli_decompress" => func!(mock_decompress),
        },
    };
    let mut wasi_ns = wasi::exports(&mut store, &func_env);
    wasi_ns.insert(
        "proc_exit",
        func!(|_: FunctionEnvMut<RunnerEnv>, code: u32| {
            Err::<(), _>(RuntimeError::new(format!("proc_exit({code})")))
        }),
    );
    imports.register_namespace("wasi_snapshot_preview1", wasi_ns);

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

    let env = func_env.as_ref(&store);
    let stdout = String::from_utf8(env.stdout.clone()).expect("utf8 stdout");

    // The guest prints "(<in> -> <out> bytes)". The mock's framing makes
    // out == in + marker exactly — a size real brotli cannot hit on the guest's
    // repetitive payload, so this proves whose output the guest saw.
    let (in_len, out_len) = parse_sizes(&stdout);
    assert!(
        stdout.contains("arb-replay brotli round-trip: ok"),
        "unexpected wasm output: {stdout:?}"
    );
    assert_eq!(
        out_len,
        in_len + MOCK_MARKER.len(),
        "guest did not see the mock's output: {stdout:?}"
    );
    assert_eq!(
        env.compress_params,
        Some((11, DEFAULT_WINDOW_SIZE, Dictionary::Empty)),
        "guest passed unexpected compress arguments"
    );
    assert_eq!(env.decompress_dictionary, Some(Dictionary::Empty));
}
