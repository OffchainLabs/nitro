//! Runtime code for replay.wasm

use crate::{
    Escape, JitConfig, STACK_SIZE,
    imports::{arbcompress, precompiles, programs, wasi_stub, wavmio},
    platform::{exit, read_input},
    stylus::{Cothread, MessageFromCothread, MessageToCothread},
};
use arbutil::{Bytes32, evm::EvmData};
use bytes::Bytes;
use corosensei::{Coroutine, CoroutineResult, Yielder, stack::DefaultStack};
use once_cell::unsync::Lazy;
use prover::{binary_input::Input, programs::meter::MeteredMachine};
use rand_pcg::Pcg32;
use std::marker::PhantomData;
use std::ops::{Deref, DerefMut};
use wasmer::{
    Engine, Function, FunctionEnv, Imports, Instance, Memory, Module, RuntimeError, Store, Value,
    imports, sys::NativeEngineExt,
};
use wasmer_vm::install_unwinder;

// Coroutine is not Send, so we cannot keep it in CustomEnvData.
// As SP1 is single-threaded, it won't hurt if we use a few static variables.
// Another way of doing this is to build a wrapper similar to SendYielder, we
// will leave it to another time to debate which is a better option.
static mut COTHREADS: Vec<Cothread> = Vec::new();
fn cothreads_mut() -> &'static mut Vec<Cothread> {
    unsafe { &mut *&raw mut COTHREADS }
}
fn cothreads() -> &'static [Cothread] {
    unsafe { &*&raw const COTHREADS }
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
    pub memory: Option<Memory>,
    pub time: u64,
    pub pcg: Pcg32,

    input: Lazy<Input>,
    yielder: SendYielder<(), MainYieldMessage>,
}

impl CustomEnvData {
    pub fn new(yielder: &Yielder<(), MainYieldMessage>) -> Self {
        // See https://github.com/OffchainLabs/nitro/blob/7e5c0bb3cfd55ef2d99abff8b3875c97f85eb1c8/arbitrator/caller-env/src/lib.rs#L27-L31
        const PCG_INIT_STATE: u64 = 0xcafef00dd15ea5e5;
        const PCG_INIT_STREAM: u64 = 0xa02bdbf7bb3c0a7;
        let pcg = Pcg32::new(PCG_INIT_STATE, PCG_INIT_STREAM);

        Self {
            memory: None,
            time: 0,
            pcg,
            input: Lazy::new(|| read_input()),
            yielder: SendYielder::new(yielder),
        }
    }

    pub fn input(&self) -> &Input {
        self.input.deref()
    }

    pub fn input_mut(&mut self) -> &mut Input {
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
        let cothread = Cothread::new(module.clone(), calldata, config, evm_data, gas);

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

        // Bound the number of loops for ease of debuging
        for _ in 0..10 {
            if queue.lock().expect("lock").peek_from_cothread().is_some() {
                return;
            }

            self.yielder.suspend(MainYieldMessage::RunLastChild);
        }
        panic!("did not receive message");
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
                let infos = module.as_sys().local_function_infos();
                // ptr => (function name, size), for precision, all usizes are casted to string
                let mut profiler_data: std::collections::HashMap<String, (String, String)> =
                    std::collections::HashMap::default();
                for (index, ptr, size) in infos {
                    if let Some(Some(name)) = mapping.get(index as usize) {
                        profiler_data.insert(ptr.to_string(), (name.clone(), size.to_string()));
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
                "brotli_compress" => func!(arbcompress::brotli_compress),
                "brotli_decompress" => func!(arbcompress::brotli_decompress),
            },
            "arbcrypto" => {
                "ecrecovery" => func!(precompiles::ecrecover),
                "keccak256" => func!(precompiles::keccak256),
            },
            "hooks" => {
                "beforeFirstIO" => func!(precompiles::dump_elf),
            },
            "wasi_snapshot_preview1" => {
                "proc_exit" => func!(wasi_stub::proc_exit),
                "sched_yield" => func!(wasi_stub::sched_yield),
                "clock_time_get" => func!(wasi_stub::clock_time_get),
                "random_get" => func!(wasi_stub::random_get),
                "poll_oneoff" => func!(wasi_stub::poll_oneoff),
                "args_sizes_get" => func!(wasi_stub::args_sizes_get),
                "args_get" => func!(wasi_stub::args_get),
                "environ_sizes_get" => func!(wasi_stub::environ_sizes_get),
                "environ_get" => func!(wasi_stub::environ_get),
                "fd_write" => func!(wasi_stub::fd_write),
                "fd_close" => func!(wasi_stub::fd_close),
                "fd_read" => func!(wasi_stub::fd_read),
                "fd_readdir" => func!(wasi_stub::fd_readdir),
                "fd_sync" => func!(wasi_stub::fd_sync),
                "fd_seek" => func!(wasi_stub::fd_seek),
                "fd_datasync" => func!(wasi_stub::fd_datasync),
                "fd_prestat_get" => func!(wasi_stub::fd_prestat_get),
                "fd_prestat_dir_name" => func!(wasi_stub::fd_prestat_dir_name),
                "fd_filestat_get" => func!(wasi_stub::fd_filestat_get),
                "fd_filestat_set_size" => func!(wasi_stub::fd_filestat_set_size),
                "fd_pread" => func!(wasi_stub::fd_pread),
                "fd_pwrite" => func!(wasi_stub::fd_pwrite),
                "fd_fdstat_get" => func!(wasi_stub::fd_fdstat_get),
                "fd_fdstat_set_flags" => func!(wasi_stub::fd_fdstat_set_flags),
                "path_open" => func!(wasi_stub::path_open),
                "path_create_directory" => func!(wasi_stub::path_create_directory),
                "path_remove_directory" => func!(wasi_stub::path_remove_directory),
                "path_readlink" => func!(wasi_stub::path_readlink),
                "path_rename" => func!(wasi_stub::path_rename),
                "path_filestat_get" => func!(wasi_stub::path_filestat_get),
                "path_unlink_file" => func!(wasi_stub::path_unlink_file),
                "sock_accept" => func!(wasi_stub::sock_accept),
                "sock_shutdown" => func!(wasi_stub::sock_shutdown),
            },
            "wavmio" => {
                "getGlobalStateBytes32" => func!(wavmio::get_global_state_bytes32),
                "setGlobalStateBytes32" => func!(wavmio::set_global_state_bytes32),
                "getGlobalStateU64" => func!(wavmio::get_global_state_u64),
                "setGlobalStateU64" => func!(wavmio::set_global_state_u64),
                "readInboxMessage" => func!(wavmio::read_inbox_message),
                "readDelayedInboxMessage" => func!(wavmio::read_delayed_inbox_message),
                "resolvePreImage" => func!(wavmio::resolve_keccak_preimage),
                "resolveTypedPreimage" => func!(wavmio::resolve_typed_preimage),
                "greedyResolveTypedPreimage" => func!(wavmio::greedy_resolve_typed_preimage),
                "validateCertificate" => func!(wavmio::validate_certificate),
            },
            "programs" => {
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
