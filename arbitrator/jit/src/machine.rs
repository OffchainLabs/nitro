// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{arbcompress, gostack::WasmEnvArc, runtime, syscall, wavmio, Opts};

use thiserror::Error;
use wasmer::{imports, Function, Instance, Module, Store, Universal};
use wasmer_compiler_cranelift::Cranelift;
use wasmer_compiler_llvm::LLVM;

use std::{io, io::BufReader, net::TcpStream, time::Instant};

pub fn create(opts: &Opts, env: WasmEnvArc) -> (Instance, WasmEnvArc) {
    let file = &opts.binary;

    let wasm = match std::fs::read(file) {
        Ok(wasm) => wasm,
        Err(err) => panic!("failed to read {}: {err}", file.to_string_lossy()),
    };

    let engine = match opts.cranelift {
        true => Universal::new(Cranelift::new()).engine(),
        false => Universal::new(LLVM::new()).engine(),
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
        Err(Self::HostIO(format!("{}", message.as_ref())))
    }
}

pub struct ProcessEnv {
    /// Whether to create child processes to handle execution
    pub forks: bool,
    /// Mechanism for asking for preimages and returning results
    pub socket: Option<(TcpStream, BufReader<TcpStream>)>,
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
            socket: None,
            last_preimage: None,
            timestamp: Instant::now(),
            reached_wavmio: false,
        }
    }
}
