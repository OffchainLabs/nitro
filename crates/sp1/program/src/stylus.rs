//! Stylus runtime. In this module we keep data structures & code
//! required to interface with wasmer. Nitro's definitions are mostly
//! kept in `nitro` sub-module.

use crate::{
    CallInputs, Escape, JitConfig, MeterData, Ptr, STACK_SIZE,
    imports::{debug, vm_hooks},
    read_bytes20, read_bytes32, read_slice,
    replay::SendYielder,
};
use arbutil::{
    Bytes20, Bytes32,
    evm::{
        ARBOS_VERSION_STYLUS_LAST_CODE_CACHE_FIX, EvmData,
        api::{EVM_API_METHOD_REQ_OFFSET, EvmApiMethod, EvmApiStatus, Gas},
        storage::{StorageCache, StorageWord},
        user::UserOutcomeKind,
    },
};
use bytes::Bytes;
use corosensei::{Coroutine, Yielder, stack::DefaultStack};
use eyre::{bail, eyre};
use prover::programs::{
    STYLUS_ENTRY_POINT,
    config::PricingParams,
    depth::STYLUS_STACK_LEFT,
    meter::{GasMeteredMachine, MachineMeter, MeteredMachine, STYLUS_INK_LEFT, STYLUS_INK_STATUS},
};
use std::collections::{VecDeque, hash_map::Entry};
use std::ops::DerefMut;
use std::sync::{Arc, Mutex};
use wasmer::{
    AsStoreMut, Engine, Function, FunctionEnv, Imports, Instance, Memory, MemoryView, Module,
    RuntimeError, Store, StoreObjects, imports, sys::NativeEngineExt,
};
use wasmer_vm::{UnwindReason, VMExtern, install_unwinder};

/// A cothread wraps a stylus program. Actually we run the stylus
/// program via a coroutine, it is just so named following existing
/// structure in arbitrator
pub struct Cothread {
    pub queue: Arc<Mutex<MessageQueue>>,
    pub coroutine: Coroutine<CothreadInput, CothreadYield, CothreadReturn>,

    store: Store,
    instance: Instance,
    function_env: FunctionEnv<StylusCustomEnvData>,
}

/// This way we can workaround Rust limitations. All variables will
/// be alive for the entire duration of Cothread, so there is no use
/// after free situation.
#[derive(Clone)]
pub struct CothreadInput {
    store: usize,
    instance: usize,
    function_env: usize,
}

pub type CothreadYield = Option<UnwindReason>;
pub type CothreadReturn = Result<u32, RuntimeError>;

impl Cothread {
    pub fn new(
        program: Bytes,
        calldata: Vec<u8>,
        config: JitConfig,
        evm_data: EvmData,
        gas: u64,
    ) -> Self {
        let queue = MessageQueue::new();
        let args_len = calldata.len();
        let mut store = Store::new(Engine::headless());
        let module = unsafe { Module::deserialize(&store, program) }.expect("creating module");
        let (imports, function_env) =
            build_imports(&mut store, calldata, config, evm_data, queue.clone());
        let instance = Instance::new(&mut store, &module, &imports).expect("instantiating module");

        let memory = instance
            .exports
            .get_memory("memory")
            .expect("fetching memory");
        function_env.as_mut(&mut store).memory = Some(memory.clone());

        // Setup ink variables
        let (ink_left, ink_status) = {
            let mut expect_global = |name| {
                let VMExtern::Global(sh) = instance
                    .exports
                    .get_extern(name)
                    .unwrap()
                    .to_vm_extern()
                    .into_sys()
                else {
                    panic!("name not found global");
                };
                let StoreObjects::Sys(objects) = store.objects_mut();
                sh.get(&objects).vmglobal()
            };
            (
                expect_global(STYLUS_INK_LEFT),
                expect_global(STYLUS_INK_STATUS),
            )
        };
        {
            let env = function_env.as_mut(&mut store);

            env.meter = Some(MeterData {
                ink_left,
                ink_status,
            });
            let ink = env.config.stylus.pricing.gas_to_ink(Gas(gas));
            env.set_ink(ink);
        }

        // Set stack left
        {
            let max_depth = {
                let env = function_env.as_mut(&mut store);
                env.config.stylus.max_depth
            };
            let Ok(global) = instance.exports.get_global(STYLUS_STACK_LEFT) else {
                panic!("global {} does not exist", STYLUS_STACK_LEFT);
            };
            global
                .set(&mut store, max_depth.into())
                .expect("set stack left")
        }

        let coroutine = Coroutine::with_stack(
            DefaultStack::new(STACK_SIZE).expect("create stylus default stack"),
            move |yielder: &Yielder<CothreadInput, CothreadYield>, input: CothreadInput| {
                let store = input.store_mut();
                let function_env = input.function_env_mut();
                let instance = input.instance();

                let send_yielder = SendYielder::new(yielder);
                {
                    let env = function_env.as_mut(store);
                    env.yielder = Some(send_yielder.clone());
                }
                install_unwinder(Some(Box::new(move |reason| {
                    send_yielder.suspend(Some(reason));
                })));

                let start = instance
                    .exports
                    .get_typed_function::<u32, u32>(&store, STYLUS_ENTRY_POINT)
                    .expect("fetching stylus entrypoint function!");

                start.call(store, args_len as u32)
            },
        );
        Self {
            queue,
            coroutine,
            store,
            instance,
            function_env,
        }
    }

    pub fn input(&self) -> CothreadInput {
        CothreadInput {
            store: (&self.store) as *const _ as usize,
            instance: (&self.instance) as *const _ as usize,
            function_env: (&self.function_env) as *const _ as usize,
        }
    }
}

impl CothreadInput {
    pub fn store_mut(&self) -> &mut Store {
        unsafe { &mut *(self.store as *mut _) }
    }

    pub fn function_env_mut(&self) -> &mut FunctionEnv<StylusCustomEnvData> {
        unsafe { &mut *(self.function_env as *mut _) }
    }

    pub fn instance(&self) -> &Instance {
        unsafe { &*(self.instance as *const _) }
    }
}

#[derive(Default)]
pub struct MessageQueue {
    tx: VecDeque<MessageToCothread>,
    rx: VecDeque<MessageFromCothread>,
}

impl MessageQueue {
    pub fn new() -> Arc<Mutex<Self>> {
        Arc::new(Mutex::new(Self::default()))
    }

    pub fn send_from_cothread(&mut self, msg: MessageFromCothread) {
        self.rx.push_back(msg);
    }

    pub fn peek_from_cothread(&self) -> Option<MessageFromCothread> {
        self.rx.front().cloned()
    }

    pub fn mark_read_from_cothread(&mut self) {
        self.rx.pop_front();
        // For now nitro uses rendezvous channel, this assertion will
        // hold.
        assert!(self.rx.is_empty());
    }

    pub fn send_to_cothread(&mut self, msg: MessageToCothread) {
        self.tx.push_back(msg);
    }

    pub fn read_to_cothread(&mut self) -> Option<MessageToCothread> {
        self.tx.pop_front()
    }
}

/// Wasmer custom env data for stylus programs
pub struct StylusCustomEnvData {
    pub memory: Option<Memory>,
    pub meter: Option<MeterData>,

    pub calldata: Vec<u8>,
    pub config: JitConfig,
    pub evm_data: EvmData,
    pub outs: Vec<u8>,
    pub storage_cache: StorageCache,

    last_return_data: Option<Vec<u8>>,
    last_code: Option<(Bytes20, Vec<u8>)>,

    queue: Arc<Mutex<MessageQueue>>,
    /// Value will be set every time current coroutine is invoked.
    pub yielder: Option<SendYielder<CothreadInput, CothreadYield>>,
}

impl StylusCustomEnvData {
    pub fn send_from_cothread(&mut self, msg: MessageFromCothread) {
        self.queue.lock().expect("lock").send_from_cothread(msg);
    }

    fn wait_next_message(&mut self) -> MessageToCothread {
        for _ in 0..10 {
            if let Some(msg) = self.queue.lock().expect("lock").read_to_cothread() {
                return msg;
            }
            self.yielder.as_ref().unwrap().suspend(None);
        }
        panic!("did not receive message to cothread");
    }

    pub fn request(
        &mut self,
        req_type: EvmApiMethod,
        req_data: Vec<u8>,
    ) -> (Vec<u8>, Vec<u8>, Gas) {
        let msg = MessageFromCothread {
            req_type: req_type as u32 + EVM_API_METHOD_REQ_OFFSET,
            req_data,
        };
        self.send_from_cothread(msg);

        let res = self.wait_next_message();
        (res.result, res.raw_data, res.cost)
    }

    pub fn get_bytes32(&mut self, key: Bytes32, evm_api_gas_to_use: Gas) -> (Bytes32, Gas) {
        let mut cost = self.storage_cache.read_gas();

        if !self.storage_cache.contains_key(&key) {
            let (res, _, gas) = self.request(EvmApiMethod::GetBytes32, key.to_vec());
            cost = cost.saturating_add(gas).saturating_add(evm_api_gas_to_use);
            self.storage_cache
                .insert(key, StorageWord::known(res.try_into().unwrap()));
        }

        (self.storage_cache[&key].value, cost)
    }

    pub fn cache_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Gas {
        let cost = self.storage_cache.write_gas();
        match self.storage_cache.entry(key) {
            Entry::Occupied(mut key) => key.get_mut().value = value,
            Entry::Vacant(slot) => drop(slot.insert(StorageWord::unknown(value))),
        };
        cost
    }

    pub fn flush_storage_cache(
        &mut self,
        clear: bool,
        gas_left: Gas,
    ) -> eyre::Result<(Gas, UserOutcomeKind)> {
        let mut data = Vec::with_capacity(64 * self.storage_cache.len() + 8);
        data.extend(gas_left.to_be_bytes());

        for (key, value) in self.storage_cache.deref_mut() {
            if value.dirty() {
                data.extend(*key);
                data.extend(*value.value);
                value.known = Some(value.value);
            }
        }
        if clear {
            self.storage_cache.clear();
        }
        if data.len() == 8 {
            return Ok((Gas(0), UserOutcomeKind::Success)); // no need to make request
        }

        let (res, _, cost) = self.request(EvmApiMethod::SetTrieSlots, data);
        let status = res.first().copied().ok_or(eyre!("empty result!"))?;
        let outcome = match status.try_into()? {
            EvmApiStatus::Success => UserOutcomeKind::Success,
            EvmApiStatus::WriteProtection => UserOutcomeKind::Revert,
            EvmApiStatus::OutOfGas => UserOutcomeKind::OutOfInk,
            _ => bail!("unexpect outcome"),
        };
        Ok((cost, outcome))
    }

    pub fn get_transient_bytes32(&mut self, key: Bytes32) -> Bytes32 {
        let (res, ..) = self.request(EvmApiMethod::GetTransientBytes32, key.to_vec());
        res.try_into().unwrap()
    }

    pub fn set_transient_bytes32(
        &mut self,
        key: Bytes32,
        value: Bytes32,
    ) -> eyre::Result<UserOutcomeKind> {
        let mut data = Vec::with_capacity(64);
        data.extend(key);
        data.extend(value);
        let (res, ..) = self.request(EvmApiMethod::SetTransientBytes32, data);
        let status = res.first().copied().ok_or(eyre!("empty result!"))?;
        let outcome = match status.try_into()? {
            EvmApiStatus::Success => UserOutcomeKind::Success,
            EvmApiStatus::WriteProtection => UserOutcomeKind::Revert,
            _ => bail!("unexpect outcome"),
        };

        Ok(outcome)
    }

    pub fn emit_log(&mut self, data: Vec<u8>, topics: u32) -> Result<(), Escape> {
        let mut request = Vec::with_capacity(4 + data.len());
        request.extend(topics.to_be_bytes());
        request.extend(data);

        let (res, _, _) = self.request(EvmApiMethod::EmitLog, request);
        if !res.is_empty() {
            return Err(String::from_utf8(res)
                .unwrap_or("malformed emit-log response".into())
                .into());
        }
        Ok(())
    }

    pub fn account_balance(&mut self, address: Bytes20) -> (Bytes32, Gas) {
        let (res, _, cost) = self.request(EvmApiMethod::AccountBalance, address.to_vec());
        (res.try_into().unwrap(), cost)
    }

    pub fn account_code(
        &mut self,
        arbos_version: u64,
        address: Bytes20,
        gas_left: Gas,
    ) -> (Vec<u8>, Gas) {
        if let Some((stored_address, data)) = self.last_code.as_ref() {
            if address == *stored_address {
                return (data.clone(), Gas(0));
            }
        }
        let mut req = Vec::with_capacity(20 + 8);
        req.extend(address);
        req.extend(gas_left.to_be_bytes());

        let (_, data, cost) = self.request(EvmApiMethod::AccountCode, req);
        if !data.is_empty() || arbos_version < ARBOS_VERSION_STYLUS_LAST_CODE_CACHE_FIX {
            self.last_code = Some((address, data.clone()));
        }
        (data, cost)
    }

    pub fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, Gas) {
        let (res, _, cost) = self.request(EvmApiMethod::AccountCodeHash, address.to_vec());
        (res.try_into().unwrap(), cost)
    }

    pub fn get_return_data(&self) -> Vec<u8> {
        self.last_return_data.clone().expect("missing return data")
    }

    fn create_request(
        &mut self,
        create_type: EvmApiMethod,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Option<Bytes32>,
        gas: Gas,
    ) -> (Result<Bytes20, String>, u32, Gas) {
        let mut request = Vec::with_capacity(8 + 2 * 32 + code.len());
        request.extend(gas.to_be_bytes());
        request.extend(endowment);
        if let Some(salt) = salt {
            request.extend(salt);
        }
        request.extend(code);

        let (mut res, data, cost) = self.request(create_type, request);
        if res.len() != 21 || res[0] == 0 {
            if !res.is_empty() {
                res.remove(0);
            }
            let err_string = String::from_utf8(res).unwrap_or("create_response_malformed".into());
            return (Err(err_string), 0, cost);
        }
        res.remove(0);
        let address = res.try_into().unwrap();
        let data_len = data.len() as u32;
        self.last_return_data = Some(data);
        (Ok(address), data_len, cost)
    }

    pub fn create1(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        gas: Gas,
    ) -> (Result<Bytes20, String>, u32, Gas) {
        self.create_request(EvmApiMethod::Create1, code, endowment, None, gas)
    }

    pub fn create2(
        &mut self,
        code: Vec<u8>,
        endowment: Bytes32,
        salt: Bytes32,
        gas: Gas,
    ) -> (Result<Bytes20, String>, u32, Gas) {
        self.create_request(EvmApiMethod::Create2, code, endowment, Some(salt), gas)
    }

    pub fn parse_call_inputs(
        &mut self,
        memory: &MemoryView,
        contract: Ptr,
        data: Ptr,
        gas: Gas,
        data_len: u32,
        value: Option<Ptr>,
    ) -> Result<CallInputs, Escape> {
        let gas_left = self.gas_left()?;
        let gas_req = gas.min(gas_left);
        let contract = read_bytes20(contract, memory)?;
        let input = read_slice(data, data_len as usize, memory)?;
        let value = value.map(|x| read_bytes32(x, memory)).transpose()?;
        Ok(CallInputs {
            contract,
            input,
            gas_left,
            gas_req,
            value,
        })
    }

    fn call_request(
        &mut self,
        call_type: EvmApiMethod,
        contract: Bytes20,
        input: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        value: Bytes32,
    ) -> (u32, Gas, UserOutcomeKind) {
        let mut request = Vec::with_capacity(20 + 32 + 8 + 8 + input.len());
        request.extend(contract);
        request.extend(value);
        request.extend(gas_left.to_be_bytes());
        request.extend(gas_req.to_be_bytes());
        request.extend(input);

        let (res, data, cost) = self.request(call_type, request);
        let status: UserOutcomeKind = res[0].try_into().expect("unknown outcome");
        let data_len = data.len() as u32;
        self.last_return_data = Some(data);
        (data_len, cost, status)
    }

    pub fn contract_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas_left: Gas,
        gas_req: Gas,
        value: Bytes32,
    ) -> (u32, Gas, UserOutcomeKind) {
        self.call_request(
            EvmApiMethod::ContractCall,
            contract,
            input,
            gas_left,
            gas_req,
            value,
        )
    }

    pub fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> (u32, Gas, UserOutcomeKind) {
        self.call_request(
            EvmApiMethod::DelegateCall,
            contract,
            input,
            gas_left,
            gas_req,
            Bytes32::default(),
        )
    }

    pub fn static_call(
        &mut self,
        contract: Bytes20,
        input: &[u8],
        gas_left: Gas,
        gas_req: Gas,
    ) -> (u32, Gas, UserOutcomeKind) {
        self.call_request(
            EvmApiMethod::StaticCall,
            contract,
            input,
            gas_left,
            gas_req,
            Bytes32::default(),
        )
    }

    pub fn add_pages(&mut self, pages: u16) -> Gas {
        self.request(EvmApiMethod::AddPages, pages.to_be_bytes().to_vec())
            .2
    }

    pub fn meter_mut(&mut self) -> &mut MeterData {
        self.meter.as_mut().expect("not metered")
    }

    pub fn meter(&self) -> &MeterData {
        self.meter.as_ref().expect("not metered")
    }
}

impl MeteredMachine for StylusCustomEnvData {
    fn ink_left(&self) -> MachineMeter {
        let vm = self.meter();
        match vm.status() {
            0 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let vm = self.meter_mut();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }
}

impl GasMeteredMachine for StylusCustomEnvData {
    fn pricing(&self) -> PricingParams {
        self.config.stylus.pricing
    }
}

fn build_imports(
    store: &mut Store,
    calldata: Vec<u8>,
    config: JitConfig,
    evm_data: EvmData,
    queue: Arc<Mutex<MessageQueue>>,
) -> (Imports, FunctionEnv<StylusCustomEnvData>) {
    let debug_funcs = config.compile.debug.debug_funcs;

    let env = StylusCustomEnvData {
        memory: None,
        meter: None,
        calldata,
        config,
        evm_data,
        outs: Vec::new(),
        storage_cache: StorageCache::default(),
        queue,
        yielder: None,
        last_code: None,
        last_return_data: None,
    };
    let func_env = FunctionEnv::new(store, env);
    macro_rules! func {
        ($func:expr) => {
            Function::new_typed_with_env(store, &func_env, $func)
        };
    }

    // TODO: this is not yet a complete list of hook APIs
    let mut imports = imports! {
        "vm_hooks" => {
            "read_args" => func!(vm_hooks::read_args),
            "write_result" => func!(vm_hooks::write_result),
            "exit_early" => func!(vm_hooks::exit_early),
            "storage_load_bytes32" => func!(vm_hooks::storage_load_bytes32),
            "storage_cache_bytes32" => func!(vm_hooks::storage_cache_bytes32),
            "storage_flush_cache" => func!(vm_hooks::storage_flush_cache),
            "transient_load_bytes32" => func!(vm_hooks::transient_load_bytes32),
            "transient_store_bytes32" => func!(vm_hooks::transient_store_bytes32),
            "call_contract" => func!(vm_hooks::call_contract),
            "delegate_call_contract" => func!(vm_hooks::delegate_call_contract),
            "static_call_contract" => func!(vm_hooks::static_call_contract),
            "create1" => func!(vm_hooks::create1),
            "create2" => func!(vm_hooks::create2),
            "read_return_data" => func!(vm_hooks::read_return_data),
            "return_data_size" => func!(vm_hooks::return_data_size),
            "emit_log" => func!(vm_hooks::emit_log),
            "account_balance" => func!(vm_hooks::account_balance),
            "account_code" => func!(vm_hooks::account_code),
            "account_codehash" => func!(vm_hooks::account_codehash),
            "account_code_size" => func!(vm_hooks::account_code_size),
            "evm_gas_left" => func!(vm_hooks::evm_gas_left),
            "evm_ink_left" => func!(vm_hooks::evm_ink_left),
            "block_basefee" => func!(vm_hooks::block_basefee),
            "chainid" => func!(vm_hooks::chainid),
            "block_coinbase" => func!(vm_hooks::block_coinbase),
            "block_gas_limit" => func!(vm_hooks::block_gas_limit),
            "block_number" => func!(vm_hooks::block_number),
            "block_timestamp" => func!(vm_hooks::block_timestamp),
            "contract_address" => func!(vm_hooks::contract_address),
            "math_div" => func!(vm_hooks::math_div),
            "math_mod" => func!(vm_hooks::math_mod),
            "math_pow" => func!(vm_hooks::math_pow),
            "math_add_mod" => func!(vm_hooks::math_add_mod),
            "math_mul_mod" => func!(vm_hooks::math_mul_mod),
            "msg_reentrant" => func!(vm_hooks::msg_reentrant),
            "msg_sender" => func!(vm_hooks::msg_sender),
            "msg_value" => func!(vm_hooks::msg_value),
            "tx_gas_price" => func!(vm_hooks::tx_gas_price),
            "tx_ink_price" => func!(vm_hooks::tx_ink_price),
            "tx_origin" => func!(vm_hooks::tx_origin),
            "pay_for_memory_grow" => func!(vm_hooks::pay_for_memory_grow),
            "native_keccak256" => func!(vm_hooks::native_keccak256),
        }
    };
    if debug_funcs {
        imports.define("console", "log_txt", func!(debug::console_log_text));
        imports.define("console", "log_i32", func!(debug::console_log::<u32>));
        imports.define("console", "log_i64", func!(debug::console_log::<u64>));
        imports.define("console", "log_f32", func!(debug::console_log::<f32>));
        imports.define("console", "log_f64", func!(debug::console_log::<f64>));
        imports.define("console", "tee_i32", func!(debug::console_tee::<u32>));
        imports.define("console", "tee_i64", func!(debug::console_tee::<u64>));
        imports.define("console", "tee_f32", func!(debug::console_tee::<f32>));
        imports.define("console", "tee_f64", func!(debug::console_tee::<f64>));
        imports.define("debug", "null_host", func!(debug::null_host));
        imports.define("debug", "start_benchmark", func!(debug::start_benchmark));
        imports.define("debug", "end_benchmark", func!(debug::end_benchmark));
    }
    (imports, func_env)
}

#[derive(Clone)]
pub struct MessageToCothread {
    pub result: Vec<u8>,
    pub raw_data: Vec<u8>,
    pub cost: Gas,
}

#[derive(Clone)]
pub struct MessageFromCothread {
    pub req_type: u32,
    pub req_data: Vec<u8>,
}
