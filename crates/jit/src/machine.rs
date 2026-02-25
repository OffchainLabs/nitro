// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{
    arbcompress, arbcrypto, prepare::prepare_env_from_json, program,
    stylus_backend::CothreadHandler, wasip1_stub, wavmio, InputMode, LocalInput, NativeInput, Opts,
    ValidatorOpts,
};
use arbutil::{Bytes32, PreimageType};
use caller_env::GoRuntimeState;
use eyre::{bail, ErrReport, Report, Result};
use sha3::{Digest, Keccak256};
use std::{
    collections::BTreeMap,
    collections::HashMap,
    fs::File,
    io::{self, BufReader, BufWriter, ErrorKind, Read},
    net::TcpStream,
    sync::Arc,
    time::Instant,
};
use thiserror::Error;
use validation::BatchInfo;
use wasmer::{
    imports, sys::CompilerConfig, Engine, Function, FunctionEnv, FunctionEnvMut, Instance, Memory,
    Module, RuntimeError, Store,
};
use wasmer_compiler_cranelift::Cranelift;

/// A pre-compiled WASM module bundled with the Engine that produced it.
///
/// Cheap to clone (both `Module` and `Engine` are Arc-based internally).
/// Safe to share across threads (`Send + Sync`).
///
/// Use [`compile_module`] to create one, then [`instantiate`] to create
/// per-request instances cheaply.
#[derive(Clone)]
pub struct CompiledModule {
    module: Module,
    engine: Engine,
}

/// Compiles a WASM binary into a reusable [`CompiledModule`].
///
/// This is the expensive operation that should be done once and cached.
/// The resulting `CompiledModule` can be passed to [`instantiate`] to create
/// per-request instances without re-compiling.
pub fn compile_module(validator: &ValidatorOpts) -> Result<CompiledModule> {
    let engine = make_engine(validator.cranelift);
    let wasm = std::fs::read(&validator.binary)?;
    let module = Module::new(&engine, wasm)?;
    Ok(CompiledModule { module, engine })
}

/// Creates a new WASM instance from a pre-compiled module and per-request options.
///
/// This is the cheap, per-request operation. It creates a fresh `Store` from the
/// compiled module's `Engine`, builds the `WasmEnv`, and instantiates the module.
pub fn instantiate(
    compiled: &CompiledModule,
    opts: &Opts,
) -> Result<(Instance, FunctionEnv<WasmEnv>, Store)> {
    let mut store = Store::new(compiled.engine.clone());

    let env = WasmEnv::try_from(opts)?;
    let func_env = FunctionEnv::new(&mut store, env);

    let imports = imports(&mut store, &func_env);
    let instance = Instance::new(&mut store, &compiled.module, &imports)?;

    let memory = instance.exports.get_memory("memory")?.clone();
    func_env.as_mut(&mut store).memory = Some(memory);

    Ok((instance, func_env, store))
}

/// Creates a WASM instance by compiling the binary and instantiating it.
///
/// This is a convenience function that combines [`compile_module`] and [`instantiate`].
/// For repeated executions with the same binary, prefer caching the result of
/// `compile_module()` and calling `instantiate()` directly.
pub fn create(opts: &Opts) -> Result<(Instance, FunctionEnv<WasmEnv>, Store)> {
    let compiled = compile_module(&opts.validator)?;
    instantiate(&compiled, opts)
}

fn make_engine(cranelift: bool) -> Engine {
    match cranelift {
        true => make_cranelift_engine(),
        false => make_llvm_engine(),
    }
}

fn make_cranelift_engine() -> Engine {
    let mut compiler = Cranelift::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();
    Engine::from(compiler)
}

#[cfg(not(feature = "llvm"))]
fn make_llvm_engine() -> Engine {
    panic!("Please rebuild with the \"llvm\" feature for LLVM support");
}
#[cfg(feature = "llvm")]
fn make_llvm_engine() -> Engine {
    let mut compiler = wasmer_compiler_llvm::LLVM::new();
    compiler.canonicalize_nans(true);
    compiler.opt_level(wasmer_compiler_llvm::LLVMOptLevel::Aggressive);
    compiler.enable_verifier();
    Engine::from(compiler)
}

fn imports(store: &mut Store, func_env: &FunctionEnv<WasmEnv>) -> wasmer::Imports {
    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(store, func_env, $func)
        };
    }
    imports! {
        "arbcompress" => {
            "brotli_compress" => func!(arbcompress::brotli_compress),
            "brotli_decompress" => func!(arbcompress::brotli_decompress),
        },
        "arbcrypto" => {
            "ecrecovery" => func!(arbcrypto::ecrecovery),
            "keccak256" => func!(arbcrypto::keccak256),
        },
        "hooks" => {
            "beforeFirstIO" => func!(|_: WasmEnvMut|{}),
        },
        "wavmio" => {
            "getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32),
            "setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32),
            "getGlobalStateU64" => func!(wavmio::get_global_state_u64),
            "setGlobalStateU64" => func!(wavmio::set_global_state_u64),
            "readInboxMessage" => func!(wavmio::read_inbox_message),
            "readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
            "resolvePreImage" => {
                #[allow(deprecated)] // we're just keeping this around until we no longer need to validate old replay binaries
                {
                    func!(wavmio::resolve_keccak_preimage)
                }
            },
            "resolveTypedPreimage" => func!(wavmio::resolve_typed_preimage),
            "validateCertificate" => func!(wavmio::validate_certificate),
        },
        "wasi_snapshot_preview1" => {
            "proc_exit" => func!(wasip1_stub::proc_exit),
            "environ_sizes_get" => func!(wasip1_stub::environ_sizes_get),
            "fd_write" => func!(wasip1_stub::fd_write),
            "environ_get" => func!(wasip1_stub::environ_get),
            "fd_close" => func!(wasip1_stub::fd_close),
            "fd_read" => func!(wasip1_stub::fd_read),
            "fd_readdir" => func!(wasip1_stub::fd_readdir),
            "fd_sync" => func!(wasip1_stub::fd_sync),
            "fd_seek" => func!(wasip1_stub::fd_seek),
            "fd_datasync" => func!(wasip1_stub::fd_datasync),
            "path_open" => func!(wasip1_stub::path_open),
            "path_create_directory" => func!(wasip1_stub::path_create_directory),
            "path_remove_directory" => func!(wasip1_stub::path_remove_directory),
            "path_readlink" => func!(wasip1_stub::path_readlink),
            "path_rename" => func!(wasip1_stub::path_rename),
            "path_filestat_get" => func!(wasip1_stub::path_filestat_get),
            "path_unlink_file" => func!(wasip1_stub::path_unlink_file),
            "fd_prestat_get" => func!(wasip1_stub::fd_prestat_get),
            "fd_prestat_dir_name" => func!(wasip1_stub::fd_prestat_dir_name),
            "fd_filestat_get" => func!(wasip1_stub::fd_filestat_get),
            "fd_filestat_set_size" => func!(wasip1_stub::fd_filestat_set_size),
            "fd_pread" => func!(wasip1_stub::fd_pread),
            "fd_pwrite" => func!(wasip1_stub::fd_pwrite),
            "sock_accept" => func!(wasip1_stub::sock_accept),
            "sock_shutdown" => func!(wasip1_stub::sock_shutdown),
            "sched_yield" => func!(wasip1_stub::sched_yield),
            "clock_time_get" => func!(wasip1_stub::clock_time_get),
            "random_get" => func!(wasip1_stub::random_get),
            "args_sizes_get" => func!(wasip1_stub::args_sizes_get),
            "args_get" => func!(wasip1_stub::args_get),
            "poll_oneoff" => func!(wasip1_stub::poll_oneoff),
            "fd_fdstat_get" => func!(wasip1_stub::fd_fdstat_get),
            "fd_fdstat_set_flags" => func!(wasip1_stub::fd_fdstat_set_flags),
        },
        "programs" => {
            "program_prepare" => func!(program::program_prepare),
            "program_requires_prepare" => func!(program::program_requires_prepare),
            "new_program" => func!(program::new_program),
            "pop" => func!(program::pop),
            "set_response" => func!(program::set_response),
            "get_request" => func!(program::get_request),
            "get_request_data" => func!(program::get_request_data),
            "start_program" => func!(program::start_program),
            "send_response" => func!(program::send_response),
            "create_stylus_config" => func!(program::create_stylus_config),
            "create_evm_data" => func!(program::create_evm_data),
            "create_evm_data_v2" => func!(program::create_evm_data_v2),
            "activate" => func!(program::activate),
            "activate_v2" => func!(program::activate_v2),
        },
    }
}

#[derive(Error, Debug)]
pub enum Escape {
    #[error("program exited with status code `{0}`")]
    Exit(u32),
    #[error("jit failed with `{0}`")]
    Failure(String),
    #[error("hostio failed with `{0}`")]
    HostIO(String),
    #[error("comms with child instance failed with `{0}`")]
    Child(ErrReport),
    #[error("hostio socket failed with `{0}`")]
    SocketError(#[from] io::Error),
    #[error("unexpected return from _start `{0:?}`")]
    UnexpectedReturn(Vec<wasmer::Value>),
}

pub type MaybeEscape = Result<(), Escape>;

impl Escape {
    pub fn exit(code: u32) -> MaybeEscape {
        Err(Self::Exit(code))
    }

    pub fn hostio<T, S: AsRef<str>>(message: S) -> Result<T, Escape> {
        Err(Self::HostIO(message.as_ref().to_string()))
    }
}

impl From<RuntimeError> for Escape {
    fn from(outcome: RuntimeError) -> Self {
        outcome
            .downcast()
            .unwrap_or_else(|outcome| Escape::Failure(format!("unknown runtime error: {outcome}")))
    }
}

pub type WasmEnvMut<'a> = FunctionEnvMut<'a, WasmEnv>;
pub type Inbox = BTreeMap<u64, Vec<u8>>;
pub type Preimages = BTreeMap<PreimageType, BTreeMap<Bytes32, Vec<u8>>>;
pub type ModuleAsm = Arc<[u8]>;

#[derive(Default)]
pub struct WasmEnv {
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Go's general runtime state
    pub go_state: GoRuntimeState,
    /// An ordered list of the 8-byte globals
    pub small_globals: [u64; 2],
    /// An ordered list of the 32-byte globals
    pub large_globals: [Bytes32; 2],
    /// An oracle allowing the prover to reverse keccak256
    pub preimages: Preimages,
    /// A collection of programs called during the course of execution
    pub module_asms: HashMap<Bytes32, ModuleAsm>,
    /// The sequencer inbox's messages
    pub sequencer_messages: Inbox,
    /// The delayed inbox's messages
    pub delayed_messages: Inbox,
    /// The purpose and connections of this process
    pub process: ProcessEnv,
    // threads
    pub threads: Vec<CothreadHandler>,
}

impl TryFrom<&Opts> for WasmEnv {
    type Error = Report;

    fn try_from(opts: &Opts) -> Result<Self> {
        let mut env = Self::default();
        env.process.debug = opts.validator.debug;

        match &opts.input_mode {
            InputMode::Json { inputs } => prepare_env_from_json(inputs, opts.validator.debug),
            InputMode::Local(local) => prepare_env_from_files(env, local),
            InputMode::Native(native) => prepare_env_from_native(env, native),
            InputMode::Continuous => Ok(env),
        }
    }
}

fn prepare_env_from_files(env: WasmEnv, input: &LocalInput) -> Result<WasmEnv> {
    let mut native = NativeInput {
        old_state: input.old_state.clone(),
        inbox: vec![],
        delayed_inbox: vec![],
        preimages: HashMap::new(),
        programs: HashMap::new(),
    };

    let mut inbox_position = input.old_state.inbox_position;
    let mut delayed_position = input.delayed_inbox_position;

    for path in &input.inbox {
        let mut msg = vec![];
        File::open(path)?.read_to_end(&mut msg)?;
        native.inbox.push(BatchInfo {
            number: inbox_position,
            data: msg,
        });
        inbox_position += 1;
    }
    for path in &input.delayed_inbox {
        let mut msg = vec![];
        File::open(path)?.read_to_end(&mut msg)?;
        native.delayed_inbox.push(BatchInfo {
            number: delayed_position,
            data: msg,
        });
        delayed_position += 1;
    }

    if let Some(path) = &input.preimages {
        let mut file = BufReader::new(File::open(path)?);
        let mut preimages = Vec::new();
        let filename = path.to_string_lossy();
        loop {
            let mut size_buf = [0u8; 8];
            match file.read_exact(&mut size_buf) {
                Ok(()) => {}
                Err(err) if err.kind() == ErrorKind::UnexpectedEof => break,
                Err(err) => bail!("Failed to parse {filename}: {}", err),
            }
            let size = u64::from_le_bytes(size_buf) as usize;
            let mut buf = vec![0u8; size];
            file.read_exact(&mut buf)?;
            preimages.push(buf);
        }
        let keccak_preimages = native.preimages.entry(PreimageType::Keccak256).or_default();
        for preimage in preimages {
            let mut hasher = Keccak256::new();
            hasher.update(&preimage);
            let hash = hasher.finalize().into();
            keccak_preimages.insert(hash, preimage);
        }
    }

    prepare_env_from_native(env, &native)
}

fn prepare_env_from_native(mut env: WasmEnv, input: &NativeInput) -> Result<WasmEnv> {
    env.process.already_has_input = true;

    for msg in &input.inbox {
        env.sequencer_messages.insert(msg.number, msg.data.clone());
    }
    for msg in &input.delayed_inbox {
        env.delayed_messages.insert(msg.number, msg.data.clone());
    }

    for (preimage_type, preimages_map) in &input.preimages {
        let type_map = env.preimages.entry(*preimage_type).or_default();
        for (hash, preimage) in preimages_map {
            type_map.insert(*hash, preimage.clone());
        }
    }

    for (hash, program) in &input.programs {
        env.module_asms.insert(*hash, program.as_ref().into());
    }

    env.small_globals = [
        input.old_state.inbox_position,
        input.old_state.position_within_message,
    ];
    env.large_globals = [
        input.old_state.last_block_hash,
        input.old_state.last_send_root,
    ];
    Ok(env)
}

pub struct ProcessEnv {
    /// Whether the validation input is already available or do we have to fork and read it
    pub already_has_input: bool,
    /// Whether to print debugging info
    pub debug: bool,
    /// Mechanism for asking for preimages and returning results
    pub socket: Option<(BufWriter<TcpStream>, BufReader<TcpStream>)>,
    /// A timestamp that helps with printing at various moments
    pub timestamp: Instant,
    /// Whether the machine has reached the first wavmio instruction
    pub reached_wavmio: bool,
}

impl Default for ProcessEnv {
    fn default() -> Self {
        Self {
            already_has_input: false,
            debug: false,
            socket: None,
            timestamp: Instant::now(),
            reached_wavmio: false,
        }
    }
}
