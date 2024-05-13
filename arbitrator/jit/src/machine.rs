// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
<<<<<<< HEAD
    arbcompress, espressocrypto, gostack::GoRuntimeState, runtime, socket, syscall,
    syscall::JsRuntimeState, wavmio, wavmio::Bytes32, Opts,
=======
    arbcompress, caller_env::GoRuntimeState, program, socket, stylus_backend::CothreadHandler,
    wasip1_stub, wavmio, Opts,
>>>>>>> 28033f9469206d8f9639023772d51882bba8883b
};
use arbutil::{Bytes32, Color, PreimageType};
use eyre::{bail, ErrReport, Result, WrapErr};
use sha3::{Digest, Keccak256};
use std::{
    collections::{BTreeMap, HashMap},
    fs::File,
    io::{self, Write},
    io::{BufReader, BufWriter, ErrorKind, Read},
    net::TcpStream,
    sync::Arc,
    time::{Duration, Instant},
};
use thiserror::Error;
use wasmer::{
    imports, CompilerConfig, Function, FunctionEnv, FunctionEnvMut, Instance, Memory, Module,
    Pages, RuntimeError, Store,
};
use wasmer_compiler_cranelift::Cranelift;

pub fn create(opts: &Opts, env: WasmEnv) -> (Instance, FunctionEnv<WasmEnv>, Store) {
    let file = &opts.binary;

    let wasm = match std::fs::read(file) {
        Ok(wasm) => wasm,
        Err(err) => panic!("failed to read {}: {err}", file.to_string_lossy()),
    };

    let mut store = match opts.cranelift {
        true => {
            let mut compiler = Cranelift::new();
            compiler.canonicalize_nans(true);
            compiler.enable_verifier();
            Store::new(compiler)
        }
        false => {
            #[cfg(not(feature = "llvm"))]
            panic!("Please rebuild with the \"llvm\" feature for LLVM support");
            #[cfg(feature = "llvm")]
            {
                let mut compiler = wasmer_compiler_llvm::LLVM::new();
                compiler.canonicalize_nans(true);
                compiler.opt_level(wasmer_compiler_llvm::LLVMOptLevel::Aggressive);
                compiler.enable_verifier();
                Store::new(compiler)
            }
        }
    };

    let module = match Module::new(&store, wasm) {
        Ok(module) => module,
        Err(err) => panic!("{}", err),
    };

    let func_env = FunctionEnv::new(&mut store, env);
    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(&mut store, &func_env, $func)
        };
    }
    let imports = imports! {
<<<<<<< HEAD
        "go" => {
            "debug" => native!(runtime::go_debug),

            "runtime.resetMemoryDataView" => native!(runtime::reset_memory_data_view),
            "runtime.wasmExit" => func!(runtime::wasm_exit),
            "runtime.wasmWrite" => func!(runtime::wasm_write),
            "runtime.nanotime1" => func!(runtime::nanotime1),
            "runtime.walltime" => func!(runtime::walltime),
            "runtime.walltime1" => func!(runtime::walltime1),
            "runtime.scheduleTimeoutEvent" => func!(runtime::schedule_timeout_event),
            "runtime.clearTimeoutEvent" => func!(runtime::clear_timeout_event),
            "runtime.getRandomData" => func!(runtime::get_random_data),

            "syscall/js.finalizeRef" => func!(syscall::js_finalize_ref),
            "syscall/js.stringVal" => func!(syscall::js_string_val),
            "syscall/js.valueGet" => func!(syscall::js_value_get),
            "syscall/js.valueSet" => func!(syscall::js_value_set),
            "syscall/js.valueDelete" => func!(syscall::js_value_delete),
            "syscall/js.valueIndex" => func!(syscall::js_value_index),
            "syscall/js.valueSetIndex" => func!(syscall::js_value_set_index),
            "syscall/js.valueCall" => func!(syscall::js_value_call),
            "syscall/js.valueInvoke" => func!(syscall::js_value_invoke),
            "syscall/js.valueNew" => func!(syscall::js_value_new),
            "syscall/js.valueLength" => func!(syscall::js_value_length),
            "syscall/js.valuePrepareString" => func!(syscall::js_value_prepare_string),
            "syscall/js.valueLoadString" => func!(syscall::js_value_load_string),
            "syscall/js.valueInstanceOf" => func!(syscall::js_value_instance_of),
            "syscall/js.copyBytesToGo" => func!(syscall::js_copy_bytes_to_go),
            "syscall/js.copyBytesToJS" => func!(syscall::js_copy_bytes_to_js),

            "github.com/offchainlabs/nitro/wavmio.getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32),
            "github.com/offchainlabs/nitro/wavmio.setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32),
            "github.com/offchainlabs/nitro/wavmio.getGlobalStateU64" => func!(wavmio::get_global_state_u64),
            "github.com/offchainlabs/nitro/wavmio.setGlobalStateU64" => func!(wavmio::set_global_state_u64),
            "github.com/offchainlabs/nitro/wavmio.readInboxMessage" => func!(wavmio::read_inbox_message),
            "github.com/offchainlabs/nitro/wavmio.readHotShotCommitment" => func!(wavmio::read_hotshot_commitment),
            "github.com/offchainlabs/nitro/wavmio.readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
            "github.com/offchainlabs/nitro/espressocrypto.verifyNamespace" => func!(espressocrypto::verify_namespace),
            "github.com/offchainlabs/nitro/espressocrypto.verifyMerkleProof" => func!(espressocrypto::verify_merkle_proof),
            "github.com/offchainlabs/nitro/wavmio.resolvePreImage" => {
=======
        "arbcompress" => {
            "brotli_compress" => func!(arbcompress::brotli_compress),
            "brotli_decompress" => func!(arbcompress::brotli_decompress),
        },
        "wavmio" => {
            "getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32),
            "setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32),
            "getGlobalStateU64" => func!(wavmio::get_global_state_u64),
            "setGlobalStateU64" => func!(wavmio::set_global_state_u64),
            "readInboxMessage" => func!(wavmio::read_inbox_message),
            "readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
            "resolvePreImage" => {
>>>>>>> 28033f9469206d8f9639023772d51882bba8883b
                #[allow(deprecated)] // we're just keeping this around until we no longer need to validate old replay binaries
                {
                    func!(wavmio::resolve_keccak_preimage)
                }
            },
            "resolveTypedPreimage" => func!(wavmio::resolve_typed_preimage),
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
            "new_program" => func!(program::new_program),
            "pop" => func!(program::pop),
            "set_response" => func!(program::set_response),
            "get_request" => func!(program::get_request),
            "get_request_data" => func!(program::get_request_data),
            "start_program" => func!(program::start_program),
            "send_response" => func!(program::send_response),
            "create_stylus_config" => func!(program::create_stylus_config),
            "create_evm_data" => func!(program::create_evm_data),
            "activate" => func!(program::activate),
        },
    };

    let instance = match Instance::new(&mut store, &module, &imports) {
        Ok(instance) => instance,
        Err(err) => panic!("Failed to create instance: {}", err.red()),
    };
    let memory = match instance.exports.get_memory("memory") {
        Ok(memory) => memory.clone(),
        Err(err) => panic!("Failed to get memory: {}", err.red()),
    };

    let env = func_env.as_mut(&mut store);
    env.memory = Some(memory);
    (instance, func_env, store)
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
}

pub type MaybeEscape = Result<(), Escape>;

impl Escape {
    pub fn exit(code: u32) -> MaybeEscape {
        Err(Self::Exit(code))
    }

    pub fn hostio<T, S: std::convert::AsRef<str>>(message: S) -> Result<T, Escape> {
        Err(Self::HostIO(message.as_ref().to_string()))
    }
}

impl From<RuntimeError> for Escape {
    fn from(outcome: RuntimeError) -> Self {
        match outcome.downcast() {
            Ok(escape) => escape,
            Err(outcome) => Escape::Failure(format!("unknown runtime error: {outcome}")),
        }
    }
}

pub type WasmEnvMut<'a> = FunctionEnvMut<'a, WasmEnv>;
pub type Inbox = BTreeMap<u64, Vec<u8>>;
<<<<<<< HEAD
pub type HotShotCommitmentMap = BTreeMap<u64, [u8; 32]>;
pub type Preimages = BTreeMap<PreimageType, BTreeMap<[u8; 32], Vec<u8>>>;
=======
pub type Preimages = BTreeMap<PreimageType, BTreeMap<Bytes32, Vec<u8>>>;
pub type ModuleAsm = Arc<[u8]>;
>>>>>>> 28033f9469206d8f9639023772d51882bba8883b

#[derive(Default)]
pub struct WasmEnv {
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Go's general runtime state
    pub go_state: GoRuntimeState,
    /// An ordered list of the 8-byte globals
    pub small_globals: [u64; 3],
    /// An ordered list of the 32-byte globals
    pub large_globals: [Bytes32; 2],
    /// An oracle allowing the prover to reverse keccak256
    pub preimages: Preimages,
    /// A collection of programs called during the course of execution
    pub module_asms: HashMap<Bytes32, ModuleAsm>,
    /// The sequencer inbox's messages
    pub sequencer_messages: Inbox,
    /// Mapping from batch positions to hotshot commitments
    pub hotshot_comm_map: HotShotCommitmentMap,
    /// The delayed inbox's messages
    pub delayed_messages: Inbox,
    /// The purpose and connections of this process
    pub process: ProcessEnv,
    // threads
    pub threads: Vec<CothreadHandler>,
}

impl WasmEnv {
    pub fn cli(opts: &Opts) -> Result<Self> {
        let mut env = WasmEnv::default();
        env.process.forks = opts.forks;
        env.process.debug = opts.debug;

        let mut inbox_position = opts.inbox_position;
        let mut delayed_position = opts.delayed_inbox_position;

        for path in &opts.inbox {
            let mut msg = vec![];
            File::open(path)?.read_to_end(&mut msg)?;
            env.sequencer_messages.insert(inbox_position, msg);
            inbox_position += 1;
        }
        for path in &opts.delayed_inbox {
            let mut msg = vec![];
            File::open(path)?.read_to_end(&mut msg)?;
            env.delayed_messages.insert(delayed_position, msg);
            delayed_position += 1;
        }

        if let Some(path) = &opts.preimages {
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
            let keccak_preimages = env.preimages.entry(PreimageType::Keccak256).or_default();
            for preimage in preimages {
                let mut hasher = Keccak256::new();
                hasher.update(&preimage);
                let hash = hasher.finalize().into();
                keccak_preimages.insert(hash, preimage);
            }
        }

        fn parse_hex(arg: &Option<String>, name: &str) -> Result<Bytes32> {
            match arg {
                Some(arg) => {
                    let mut arg = arg.as_str();
                    if arg.starts_with("0x") {
                        arg = &arg[2..];
                    }
                    let mut bytes32 = [0u8; 32];
                    hex::decode_to_slice(arg, &mut bytes32)
                        .wrap_err_with(|| format!("failed to parse {} contents", name))?;
                    Ok(bytes32.into())
                }
                None => Ok(Bytes32::default()),
            }
        }

        let last_block_hash = parse_hex(&opts.last_block_hash, "--last-block-hash")?;
        let last_send_root = parse_hex(&opts.last_send_root, "--last-send-root")?;
        env.small_globals = [opts.inbox_position, opts.position_within_message, 0];
        env.large_globals = [last_block_hash, last_send_root];
        Ok(env)
    }

    pub fn send_results(&mut self, error: Option<String>, memory_used: Pages) {
        let writer = match &mut self.process.socket {
            Some((writer, _)) => writer,
            None => return,
        };

        macro_rules! check {
            ($expr:expr) => {{
                if let Err(comms_error) = $expr {
                    eprintln!("Failed to send results to Go: {comms_error}");
                    panic!("Communication failure");
                }
            }};
        }

        if let Some(error) = error {
            check!(socket::write_u8(writer, socket::FAILURE));
            check!(socket::write_bytes(writer, &error.into_bytes()));
            check!(writer.flush());
            return;
        }

        check!(socket::write_u8(writer, socket::SUCCESS));
        check!(socket::write_u64(writer, self.small_globals[0]));
        check!(socket::write_u64(writer, self.small_globals[1]));
        check!(socket::write_u64(writer, self.small_globals[2]));
        check!(socket::write_bytes32(writer, &self.large_globals[0]));
        check!(socket::write_bytes32(writer, &self.large_globals[1]));
        check!(socket::write_u64(writer, memory_used.bytes().0 as u64));
        check!(writer.flush());
    }
}

pub struct ProcessEnv {
    /// Whether to create child processes to handle execution
    pub forks: bool,
    /// Whether to print debugging info
    pub debug: bool,
    /// Mechanism for asking for preimages and returning results
    pub socket: Option<(BufWriter<TcpStream>, BufReader<TcpStream>)>,
    /// A timestamp that helps with printing at various moments
    pub timestamp: Instant,
    /// How long to wait on any child threads to compute a result
    pub child_timeout: Duration,
    /// Whether the machine has reached the first wavmio instruction
    pub reached_wavmio: bool,
}

impl Default for ProcessEnv {
    fn default() -> Self {
        Self {
            forks: false,
            debug: false,
            socket: None,
            timestamp: Instant::now(),
            child_timeout: Duration::from_secs(15),
            reached_wavmio: false,
        }
    }
}
