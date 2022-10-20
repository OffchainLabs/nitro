// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    arbcompress, gostack::GoRuntimeState, runtime, socket, syscall, syscall::JsRuntimeState,
    wavmio, wavmio::Bytes32, Opts,
};

use eyre::{bail, Result, WrapErr};
use parking_lot::Mutex;
use sha3::{Digest, Keccak256};
use thiserror::Error;
use wasmer::{
    imports, CompilerConfig, Function, Instance, LazyInit, Memory, Module, NativeFunc,
    RuntimeError, Store, Universal, WasmerEnv,
};
use wasmer_compiler_cranelift::Cranelift;

use std::{
    collections::BTreeMap,
    fs::File,
    io::{self, Write},
    io::{BufReader, BufWriter, ErrorKind, Read},
    net::TcpStream,
    ops::Deref,
    sync::Arc,
    time::Instant,
};

pub fn create(opts: &Opts, env: WasmEnvArc) -> (Instance, WasmEnvArc) {
    let file = &opts.binary;

    let wasm = match std::fs::read(file) {
        Ok(wasm) => wasm,
        Err(err) => panic!("failed to read {}: {err}", file.to_string_lossy()),
    };

    let engine = match opts.cranelift {
        true => {
            let mut compiler = Cranelift::new();
            compiler.canonicalize_nans(true);
            compiler.enable_verifier();
            Universal::new(compiler).engine()
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
                Universal::new(compiler).engine()
            }
        }
    };

    let store = Store::new(&engine);
    let module = match Module::new(&store, &wasm) {
        Ok(module) => module,
        Err(err) => panic!("{}", err),
    };

    macro_rules! native {
        ($func:expr) => {
            Function::new_native(&store, $func)
        };
    }
    macro_rules! func {
        ($func:expr) => {
            Function::new_native_with_env(&store, env.clone(), $func)
        };
    }

    let imports = imports! {
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
            "github.com/offchainlabs/nitro/wavmio.readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
            "github.com/offchainlabs/nitro/wavmio.resolvePreImage" => func!(wavmio::resolve_preimage),

            "github.com/offchainlabs/nitro/arbcompress.brotliCompress" => func!(arbcompress::brotli_compress),
            "github.com/offchainlabs/nitro/arbcompress.brotliDecompress" => func!(arbcompress::brotli_decompress),
        },
    };

    let instance = match Instance::new(&module, &imports) {
        Ok(instance) => instance,
        Err(err) => panic!("Failed to create instance: {}", err),
    };
    let memory = match instance.exports.get_memory("mem") {
        Ok(memory) => memory.clone(),
        Err(err) => panic!("Failed to get memory: {}", err),
    };
    env.lock().memory = Some(memory);
    (instance, env)
}

#[derive(Error, Debug)]
pub enum Escape {
    #[error("program exited with status code `{0}`")]
    Exit(u32),
    #[error("jit failed with `{0}`")]
    Failure(String),
    #[error("hostio failed with `{0}`")]
    HostIO(String),
    #[error("hostio socket failed with `{0}`")]
    SocketError(#[from] io::Error),
}

pub type MaybeEscape = Result<(), Escape>;

impl Escape {
    pub fn exit(code: u32) -> MaybeEscape {
        Err(Self::Exit(code))
    }

    pub fn hostio<S: std::convert::AsRef<str>>(message: S) -> MaybeEscape {
        Err(Self::HostIO(message.as_ref().to_string()))
    }

    pub fn failure<S: std::convert::AsRef<str>>(message: S) -> MaybeEscape {
        Err(Self::Failure(message.as_ref().to_string()))
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

pub type Inbox = BTreeMap<u64, Vec<u8>>;
pub type Oracle = BTreeMap<[u8; 32], Vec<u8>>;

#[derive(Default)]
pub struct WasmEnv {
    /// Mechanism for reading and writing the module's memory
    pub memory: Option<Memory>,
    /// Go's general runtime state
    pub go_state: GoRuntimeState,
    /// The state of Go's js runtime
    pub js_state: JsRuntimeState,
    /// An ordered list of the 8-byte globals
    pub small_globals: [u64; 2],
    /// An ordered list of the 32-byte globals
    pub large_globals: [Bytes32; 2],
    /// An oracle allowing the prover to reverse keccak256
    pub preimages: Oracle,
    /// The sequencer inbox's messages
    pub sequencer_messages: Inbox,
    /// The delayed inbox's messages
    pub delayed_messages: Inbox,
    /// The purpose and connections of this process
    pub process: ProcessEnv,
}

#[derive(Clone, Default, WasmerEnv)]
pub struct WasmEnvArc {
    env: Arc<Mutex<WasmEnv>>,
    #[wasmer(export(name = "resume"))]
    resume: LazyInit<NativeFunc<(), ()>>,
    #[wasmer(export(name = "getsp"))]
    get_stack_pointer: LazyInit<NativeFunc<(), i32>>,
}

impl Deref for WasmEnvArc {
    type Target = Mutex<WasmEnv>;
    fn deref(&self) -> &Self::Target {
        &*self.env
    }
}

impl WasmEnvArc {
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
            for preimage in preimages {
                let mut hasher = Keccak256::new();
                hasher.update(&preimage);
                let hash = hasher.finalize().into();
                env.preimages.insert(hash, preimage);
            }
        }

        fn parse_hex(arg: &Option<String>, name: &str) -> Result<Bytes32> {
            match arg {
                Some(arg) => {
                    let mut arg = arg.as_str();
                    if arg.starts_with("0x") {
                        arg = &arg[2..];
                    }
                    let mut bytes32 = Bytes32::default();
                    hex::decode_to_slice(arg, &mut bytes32)
                        .wrap_err_with(|| format!("failed to parse {} contents", name))?;
                    Ok(bytes32)
                }
                None => Ok(Bytes32::default()),
            }
        }

        let last_block_hash = parse_hex(&opts.last_block_hash, "--last-block-hash")?;
        let last_send_root = parse_hex(&opts.last_send_root, "--last-send-root")?;
        env.small_globals = [opts.inbox_position, opts.position_within_message];
        env.large_globals = [last_block_hash, last_send_root];
        let env = Arc::new(Mutex::new(env));
        let resume = LazyInit::new();
        let get_stack_pointer = LazyInit::new();
        Ok(Self {
            env,
            resume,
            get_stack_pointer,
        })
    }

    pub fn send_results(self, error: Option<String>) {
        let env = &mut *self.lock();

        let writer = match &mut env.process.socket {
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
        check!(socket::write_u64(writer, env.small_globals[0]));
        check!(socket::write_u64(writer, env.small_globals[1]));
        check!(socket::write_bytes32(writer, &env.large_globals[0]));
        check!(socket::write_bytes32(writer, &env.large_globals[1]));
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
    /// The last preimage received over the socket
    pub last_preimage: Option<([u8; 32], Vec<u8>)>,
    /// A timestamp that helps with printing at various moments
    pub timestamp: Instant,
    /// Whether the machine has reached the first wavmio instruction
    pub reached_wavmio: bool,
}

impl Default for ProcessEnv {
    fn default() -> Self {
        Self {
            forks: false,
            debug: false,
            socket: None,
            last_preimage: None,
            timestamp: Instant::now(),
            reached_wavmio: false,
        }
    }
}
