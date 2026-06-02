//! Runtime code for replay.wasm

use std::{
    marker::PhantomData,
    ops::{Deref, DerefMut},
};

use arbutil::{Bytes32, evm::EvmData};
use bytes::Bytes;
use caller_env::{
    GoRuntimeState,
    arbcrypto::host::{ecrecovery, keccak256},
    brotli::host::{brotli_compress, brotli_decompress},
    wasip1_stub::host as wasi,
    wavmio::host as wavmio,
};
use corosensei::{Coroutine, CoroutineResult, Yielder, stack::DefaultStack};
use once_cell::unsync::Lazy;
use prover::programs::meter::MeteredMachine;
use validation::ValidationInput;
use wasmer::{
    Engine, Function, FunctionEnv, FunctionEnvMut, Imports, Instance, Memory, Module, RuntimeError,
    Store, Value, imports, sys::NativeEngineExt,
};
use wasmer_vm::install_unwinder;

use crate::{
    Escape, JitConfig, STACK_SIZE,
    imports::{programs},
    platform,
    platform::{exit, read_input},
    stylus::{Cothread, MessageFromCothread, MessageToCothread},
};

// Coroutine is not Send, so we cannot keep it in CustomEnvData.
// As SP1 is single-threaded, it won't hurt if we use a few static variables.
// Another way of doing this is to build a wrapper similar to SendYielder, we
// will leave it to another time to debate which is a better option.
static mut COTHREADS: Vec<Cothread> = Vec::new();
fn cothreads_mut() -> &'static mut Vec<Cothread> {
    unsafe { &mut *std::ptr::addr_of_mut!(COTHREADS) }
}
fn cothreads() -> &'static [Cothread] {
    unsafe { &*std::ptr::addr_of!(COTHREADS) }
}

/// This provides a single-threaded Send yielder since corosensei's
/// own Yielder does not implement Send
pub struct SendYielder<Input, Yield> {
    yielder: u64,
    _input: PhantomData<Input>,
    _yield: PhantomData<Yield>,
}

impl<Input, Yield> Clone for SendYielder<Input, Yield> {
    fn clone(&self) -> Self {
        Self {
            yielder: self.yielder,
            _input: PhantomData,
            _yield: PhantomData,
        }
    }
}

impl<Input, Yield> std::ops::Deref for SendYielder<Input, Yield> {
    type Target = Yielder<Input, Yield>;

    fn deref(&self) -> &Self::Target {
        self.yielder()
    }
}

impl<Input, Yield> SendYielder<Input, Yield> {
    pub fn new(yielder: &Yielder<Input, Yield>) -> Self {
        Self {
            yielder: yielder as *const _ as u64,
            _input: PhantomData,
            _yield: PhantomData,
        }
    }

    pub fn yielder(&self) -> &Yielder<Input, Yield> {
        unsafe { &*(self.yielder as *const _) }
    }
}

#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub enum MainYieldMessage {
    RunLastChild,
}

pub struct CustomEnvData {
    /// Note this is an option, since memory is not available when building
    /// imports. A multi-step solution is required for initialization:
    ///
    /// * Build imports with memory set to None
    /// * Use imports to initialize Instance
    /// * Extract memory from instance's exports
    /// * Set the memory back in CustomEnvData.
    memory: Option<Memory>,
    pub go_state: GoRuntimeState,

    input: Lazy<ValidationInput>,
    yielder: SendYielder<(), MainYieldMessage>,
}

impl caller_env::wasmer_traits::HasMemory for CustomEnvData {
    fn memory(&self) -> Memory {
        self.memory
            .clone()
            .expect("memory not set in CustomEnvData")
    }
}

impl CustomEnvData {
    pub fn new(yielder: &Yielder<(), MainYieldMessage>) -> Self {
        Self {
            memory: None,
            go_state: GoRuntimeState::default(),
            input: Lazy::new(read_input),
            yielder: SendYielder::new(yielder),
        }
    }

    pub fn input(&self) -> &ValidationInput {
        self.input.deref()
    }

    pub fn input_mut(&mut self) -> &mut ValidationInput {
        self.input.deref_mut()
    }

    pub fn input_initialized(&self) -> bool {
        Lazy::get(&self.input).is_some()
    }

    pub fn launch_program(
        &mut self,
        module_hash: &Bytes32,
        calldata: Vec<u8>,
        config: JitConfig,
        evm_data: EvmData,
        gas: u64,
    ) -> Result<u32, Escape> {
        let Some(module) = self.input.module_asms.get(module_hash.deref()) else {
            return Escape::logical(format!("Unable to locate module: {module_hash}"));
        };
        let aligned = align_bytes(module);
        let cothread = Cothread::new(aligned, calldata, config, evm_data, gas);

        cothreads_mut().push(cothread);
        Ok(cothreads().len().try_into().unwrap())
    }

    pub fn send_to_cothread(&mut self, msg: MessageToCothread) {
        let queue = &cothreads().last().unwrap().queue;
        queue.lock().expect("lock").send_to_cothread(msg);
    }

    pub fn wait_next_message(&mut self, module: Option<u32>) {
        if let Some(module) = module {
            assert_ne!(module, 0);
            assert_eq!(module, cothreads().len() as u32);
        }

        let queue = &cothreads().last().unwrap().queue;
        queue.lock().expect("lock").mark_read_from_cothread();

        // Bound the number of loops for ease of debugging.
        // 10 iterations should be more than enough for any single message exchange.
        const MAX_YIELD_ITERATIONS: usize = 10;
        for _ in 0..MAX_YIELD_ITERATIONS {
            if queue.lock().expect("lock").peek_from_cothread().is_some() {
                return;
            }

            self.yielder.suspend(MainYieldMessage::RunLastChild);
        }
        panic!(
            "did not receive message from cothread after {MAX_YIELD_ITERATIONS} iterations (module={module:?}, num_cothreads={})",
            cothreads().len()
        );
    }

    // For now, message id in arbitrator is hardcoded to 0x33333333,
    // we are safely ignoring it
    pub fn get_last_msg(&self) -> MessageFromCothread {
        let queue = &cothreads().last().unwrap().queue;
        queue
            .lock()
            .expect("lock")
            .peek_from_cothread()
            .expect("no message waiting")
    }

    pub fn pop_last_program(&mut self) {
        cothreads_mut().pop();
    }
}

/// Given replay.wasm's serialized module(or serialized object), this method starts the main
/// event loop.
pub fn run(m: Bytes) -> ! {
    // Runs the wasmer module in a coroutine, so we can multiplex between different
    // modules without threads.
    let mut coro = Coroutine::with_stack(
        DefaultStack::new(STACK_SIZE).expect("create default stack"),
        |yielder: &Yielder<(), MainYieldMessage>, ()| {
            let mut store = Store::new(Engine::headless());
            let module = unsafe { Module::deserialize(&store, m) }.expect("creating module");

            // Setup replay.wasm function symbols for profiling & debugging
            #[cfg(target_os = "zkvm")]
            {
                let sp1_zkvm::ReadVecResult { ptr, len, .. } = sp1_zkvm::read_vec_raw();
                assert!(!ptr.is_null());
                let mapping_bytes = unsafe { std::slice::from_raw_parts(ptr, len) };
                let mapping: Vec<Option<String>> =
                    serde_json::from_slice(&mapping_bytes[..]).expect("parse mapping");
                let artifact = module.sys_artifact().expect("sys artifact");
                let extents = artifact
                    .finished_function_extents()
                    .expect("function extents");
                // ptr => (function name, size), for precision, all usizes are casted to string
                let mut profiler_data: std::collections::HashMap<String, (String, String)> =
                    std::collections::HashMap::default();
                for (index, extent) in &extents {
                    if let Some(Some(name)) = mapping.get(index.as_u32() as usize) {
                        let ptr = *extent.ptr as usize;
                        profiler_data
                            .insert(ptr.to_string(), (name.clone(), extent.length.to_string()));
                    }
                }
                let profiler_data_str =
                    serde_json::to_string(&profiler_data).expect("profiler data to json");
                sp1_zkvm::syscalls::syscall_insert_profiler_symbols(
                    profiler_data_str.as_str().as_ptr(),
                    profiler_data_str.as_str().len() as u64,
                );
            }

            let (imports, function_env) = build_imports(&mut store, yielder);
            let instance =
                Instance::new(&mut store, &module, &imports).expect("instantiating module");

            let memory = instance
                .exports
                .get_memory("memory")
                .expect("fetching memory");
            function_env.as_mut(&mut store).memory = Some(memory.clone());

            let start = instance
                .exports
                .get_function("_start")
                .expect("fetching start function!");

            start.call(&mut store, &[])
        },
    );

    let result = loop {
        install_unwinder(None);
        match coro.resume(()) {
            CoroutineResult::Yield(msg) => match msg {
                MainYieldMessage::RunLastChild => {
                    let cothread = cothreads_mut().last_mut().unwrap();
                    let input = cothread.input();
                    let store = input.store_mut();
                    let function_env = input.function_env_mut();
                    let env = function_env.as_mut(store);
                    {
                        if let Some(yielder) = &env.yielder {
                            let yielder = yielder.clone();
                            install_unwinder(Some(Box::new(move |reason| {
                                yielder.suspend(Some(reason));
                            })));
                        }
                    }
                    store.force_create();
                    let exit = match cothread.coroutine.resume(input.clone()) {
                        CoroutineResult::Yield(y) => match y {
                            Some(unwind_reason) => {
                                unsafe {
                                    cothread.coroutine.force_reset();
                                }
                                Some(Err(unwind_reason.into_trap().into()))
                            }
                            None => None,
                        },
                        CoroutineResult::Return(r) => Some(r),
                    };
                    store.force_clean();
                    if let Some(result) = exit {
                        let env = function_env.as_mut(store);
                        let (req_type, req_data) = {
                            let (req_type, data) = match result {
                                // Success
                                Ok(0) => (0, env.outs.clone()),
                                // Revert
                                Ok(_) => (1, env.outs.clone()),
                                // Failure
                                Err(e) => match e.downcast::<Escape>() {
                                    Ok(escape) => match escape {
                                        Escape::Exit(0) => (0, env.outs.clone()),
                                        Escape::Exit(_) => (1, env.outs.clone()),
                                        _ => (2, format!("{escape:?}").as_bytes().to_vec()),
                                    },
                                    Err(e) => (2, format!("{e:?}").as_bytes().to_vec()),
                                },
                            };
                            let mut output = Vec::with_capacity(8 + data.len());
                            let ink_left = env.ink_left().into();
                            let gas_left = env.config.stylus.pricing.ink_to_gas(ink_left);
                            output.extend(gas_left.to_be_bytes());
                            output.extend(data);
                            (req_type, output)
                        };
                        let msg = MessageFromCothread { req_data, req_type };
                        env.send_from_cothread(msg);
                    }
                }
            },
            CoroutineResult::Return(result) => break result,
        }
    };
    handle_result(result);
}

fn build_imports(
    store: &mut Store,
    yielder: &Yielder<(), MainYieldMessage>,
) -> (Imports, FunctionEnv<CustomEnvData>) {
    let func_env = FunctionEnv::new(store, CustomEnvData::new(yielder));
    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(store, &func_env, $func)
        };
    }

    (
        imports! {
            "arbcompress" => {
                "brotli_compress" => func!(brotli_compress::<CustomEnvData>),
                "brotli_decompress" => func!(brotli_decompress::<CustomEnvData>),
            },
            "arbcrypto" => {
                "ecrecovery" => func!(ecrecovery::<CustomEnvData>),
                "keccak256" => func!(keccak256::<CustomEnvData>),
            },
            "hooks" => {
                "beforeFirstIO" => func!(dump_elf),
            },
            "wasi_snapshot_preview1" => {
                "proc_exit" => func!(proc_exit),
                "sched_yield" => func!(wasi::sched_yield::<CustomEnvData>),
                "clock_time_get" => func!(wasi::clock_time_get::<CustomEnvData>),
                "random_get" => func!(wasi::random_get::<CustomEnvData>),
                "poll_oneoff" => func!(wasi::poll_oneoff::<CustomEnvData>),
                "args_sizes_get" => func!(wasi::args_sizes_get::<CustomEnvData>),
                "args_get" => func!(wasi::args_get::<CustomEnvData>),
                "environ_sizes_get" => func!(wasi::environ_sizes_get::<CustomEnvData>),
                "environ_get" => func!(wasi::environ_get::<CustomEnvData>),
                "fd_write" => func!(wasi::fd_write::<CustomEnvData>),
                "fd_close" => func!(wasi::fd_close::<CustomEnvData>),
                "fd_read" => func!(wasi::fd_read::<CustomEnvData>),
                "fd_readdir" => func!(wasi::fd_readdir::<CustomEnvData>),
                "fd_sync" => func!(wasi::fd_sync::<CustomEnvData>),
                "fd_seek" => func!(wasi::fd_seek::<CustomEnvData>),
                "fd_datasync" => func!(wasi::fd_datasync::<CustomEnvData>),
                "fd_prestat_get" => func!(wasi::fd_prestat_get::<CustomEnvData>),
                "fd_prestat_dir_name" => func!(wasi::fd_prestat_dir_name::<CustomEnvData>),
                "fd_filestat_get" => func!(wasi::fd_filestat_get::<CustomEnvData>),
                "fd_filestat_set_size" => func!(wasi::fd_filestat_set_size::<CustomEnvData>),
                "fd_pread" => func!(wasi::fd_pread::<CustomEnvData>),
                "fd_pwrite" => func!(wasi::fd_pwrite::<CustomEnvData>),
                "fd_fdstat_get" => func!(wasi::fd_fdstat_get::<CustomEnvData>),
                "fd_fdstat_set_flags" => func!(wasi::fd_fdstat_set_flags::<CustomEnvData>),
                "path_open" => func!(wasi::path_open::<CustomEnvData>),
                "path_create_directory" => func!(wasi::path_create_directory::<CustomEnvData>),
                "path_remove_directory" => func!(wasi::path_remove_directory::<CustomEnvData>),
                "path_readlink" => func!(wasi::path_readlink::<CustomEnvData>),
                "path_rename" => func!(wasi::path_rename::<CustomEnvData>),
                "path_filestat_get" => func!(wasi::path_filestat_get::<CustomEnvData>),
                "path_unlink_file" => func!(wasi::path_unlink_file::<CustomEnvData>),
                "sock_accept" => func!(wasi::sock_accept::<CustomEnvData>),
                "sock_shutdown" => func!(wasi::sock_shutdown::<CustomEnvData>),
            },
            "wavmio" => {
                "getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32::<CustomEnvData>),
                "setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32::<CustomEnvData>),
                "getGlobalStateU64" => func!(wavmio::get_global_state_u64::<CustomEnvData>),
                "setGlobalStateU64" => func!(wavmio::set_global_state_u64::<CustomEnvData>),
                "readInboxMessage" => func!(wavmio::read_inbox_message::<CustomEnvData>),
                "readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message::<CustomEnvData>),
                "resolvePreImage" => func!(wavmio::resolve_keccak_preimage::<CustomEnvData>),
                "resolveTypedPreimage" => func!(wavmio::resolve_typed_preimage::<CustomEnvData>),
                "validateCertificate" => func!(wavmio::validate_certificate::<CustomEnvData>),
            },
            "programs" => {
                "program_prepare" => func!(programs::program_prepare),
                "program_requires_prepare" => func!(programs::program_requires_prepare),
                "new_program" => func!(programs::new_program),
                "pop" => func!(programs::pop),
                "set_response" => func!(programs::set_response),
                "get_request" => func!(programs::get_request),
                "get_request_data" => func!(programs::get_request_data),
                "start_program" => func!(programs::start_program),
                "send_response" => func!(programs::send_response),
                "create_stylus_config" => func!(programs::create_stylus_config),
                "create_evm_data" => func!(programs::create_evm_data),
                "create_evm_data_v2" => func!(programs::create_evm_data_v2),
                "activate" => func!(programs::activate),
                "activate_v2" => func!(programs::activate_v2),
            },
        },
        func_env,
    )
}

fn dump_elf(mut ctx: FunctionEnvMut<CustomEnvData>) {
    let data = ctx.data_mut();
    assert!(!data.input_initialized());
    platform::dump_elf();
}

fn proc_exit(mut ctx: FunctionEnvMut<CustomEnvData>, code: u32) {
    if code == 0 {
        let (data, _) = ctx.data_and_store_mut();
        platform::print_string(
            1,
            format!(
                "Validation succeeds with hash 0x{}",
                hex::encode(data.input().large_globals[0])
            )
            .as_bytes(),
        );
    }
    exit(code);
}

/// Copies `data` into 8-byte-aligned memory and returns it as `Bytes`.
/// SP1's wasmer fork requires aligned memory for `Module::deserialize`.
fn align_bytes(data: &[u8]) -> Bytes {
    use bytes::BytesMut;
    let mut buffer = BytesMut::zeroed(data.len() + 7);
    let p = buffer.as_ptr() as usize;
    let aligned_p = p.div_ceil(8) * 8;
    let offset = aligned_p - p;
    buffer[offset..offset + data.len()].copy_from_slice(data);
    let bytes = buffer.freeze();
    bytes.slice(offset..offset + data.len())
}

pub(crate) fn handle_result(result: Result<Box<[Value]>, RuntimeError>) -> ! {
    let message = match result {
        Ok(value) => format!("Machine exited prematurely with: {:?}", value),
        Err(e) => format!("Runtime error: {}", e),
    };

    if !message.is_empty() {
        println!("{message}");
    }
    exit(1);
}
