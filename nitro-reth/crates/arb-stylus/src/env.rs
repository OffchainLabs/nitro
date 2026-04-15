use std::marker::PhantomData;

use arbos::programs::types::EvmData;
use wasmer::{FunctionEnvMut, Global, Memory, MemoryView, Pages, StoreMut, Value};

use crate::{
    config::{CompileConfig, StylusConfig},
    error::Escape,
    evm_api::EvmApi,
    ink::Ink,
    meter::{GasMeteredMachine, MachineMeter, MeteredMachine, HOSTIO_INK},
};

pub type WasmEnvMut<'a, E> = FunctionEnvMut<'a, WasmEnv<E>>;

/// The WASM execution environment.
///
/// Contains all state needed during Stylus program execution,
/// including the EVM API bridge, metering state, and I/O buffers.
#[derive(Debug)]
pub struct WasmEnv<E: EvmApi> {
    /// The instance's arguments.
    pub args: Vec<u8>,
    /// The instance's return data.
    pub outs: Vec<u8>,
    /// WASM linear memory.
    pub memory: Option<Memory>,
    /// Ink metering state via WASM globals.
    pub meter: Option<MeterData>,
    /// Bridge to EVM state (storage, calls, etc.).
    pub evm_api: E,
    /// EVM context data (block info, sender, etc.).
    pub evm_data: EvmData,
    /// Cached length of the last call's return data.
    pub evm_return_data_len: u32,
    /// Compile-time configuration.
    pub compile: CompileConfig,
    /// Runtime configuration (set when running).
    pub config: Option<StylusConfig>,
    /// WASM global for ink metering (set after instantiation).
    pub ink_global: Option<Global>,
    /// WASM global for ink status (set after instantiation).
    pub ink_status_global: Option<Global>,
    _phantom: PhantomData<E>,
}

impl<E: EvmApi> WasmEnv<E> {
    pub fn new(
        compile: CompileConfig,
        config: Option<StylusConfig>,
        evm_api: E,
        evm_data: EvmData,
    ) -> Self {
        Self {
            compile,
            config,
            evm_api,
            evm_data,
            args: vec![],
            outs: vec![],
            memory: None,
            meter: None,
            ink_global: None,
            ink_status_global: None,
            evm_return_data_len: 0,
            _phantom: PhantomData,
        }
    }

    /// Create a HostioInfo and charge the standard hostio cost plus `ink`.
    pub fn start<'a>(
        env: &'a mut WasmEnvMut<'_, E>,
        ink: Ink,
    ) -> Result<HostioInfo<'a, E>, Escape> {
        let mut info = Self::program(env)?;
        info.buy_ink(HOSTIO_INK.saturating_add(ink))?;
        Ok(info)
    }

    /// Create a HostioInfo for accessing host functionality.
    pub fn program<'a>(env: &'a mut WasmEnvMut<'_, E>) -> Result<HostioInfo<'a, E>, Escape> {
        let (env, store) = env.data_and_store_mut();
        let memory = env.memory.clone().expect("WASM memory not initialized");
        let mut info = HostioInfo {
            env,
            memory,
            store,
            start_ink: Ink(0),
        };
        if info.env.evm_data.tracing {
            info.start_ink = info.ink_ready()?;
        }
        Ok(info)
    }

    pub fn meter_mut(&mut self) -> &mut MeterData {
        self.meter.as_mut().expect("not metered")
    }

    pub fn meter(&self) -> &MeterData {
        self.meter.as_ref().expect("not metered")
    }
}

/// Ink meter data stored as raw pointers to WASM globals.
///
/// These point into the wasmer Store's global storage and are used
/// for fast ink reads/writes without going through the full wasmer API.
#[derive(Clone, Copy, Debug)]
pub struct MeterData {
    ink_left: u64,
    ink_status: u32,
}

impl MeterData {
    pub fn new() -> Self {
        Self {
            ink_left: 0,
            ink_status: 0,
        }
    }

    pub fn ink(&self) -> Ink {
        Ink(self.ink_left)
    }

    pub fn status(&self) -> u32 {
        self.ink_status
    }

    pub fn set_ink(&mut self, ink: Ink) {
        self.ink_left = ink.0;
    }

    pub fn set_status(&mut self, status: u32) {
        self.ink_status = status;
    }
}

unsafe impl Send for MeterData {}

/// Wrapper providing access to host I/O operations during WASM execution.
///
/// Bundles the WasmEnv, Memory, and Store together for convenient access
/// in host function implementations.
pub struct HostioInfo<'a, E: EvmApi> {
    pub env: &'a mut WasmEnv<E>,
    pub memory: Memory,
    pub store: StoreMut<'a>,
    pub start_ink: Ink,
}

impl<E: EvmApi> HostioInfo<'_, E> {
    pub fn config(&self) -> StylusConfig {
        self.env.config.expect("no config")
    }

    pub fn pricing(&self) -> crate::config::PricingParams {
        self.config().pricing
    }

    pub fn view(&self) -> MemoryView<'_> {
        self.memory.view(&self.store)
    }

    pub fn memory_size(&self) -> Pages {
        self.memory.ty(&self.store).minimum
    }

    pub fn read_fixed<const N: usize>(
        &self,
        ptr: u32,
    ) -> Result<[u8; N], wasmer::MemoryAccessError> {
        let mut data = [0u8; N];
        self.view().read(ptr as u64, &mut data)?;
        Ok(data)
    }

    pub fn read_slice(&self, ptr: u32, len: u32) -> Result<Vec<u8>, wasmer::MemoryAccessError> {
        let mut data = vec![0u8; len as usize];
        self.view().read(ptr as u64, &mut data)?;
        Ok(data)
    }

    pub fn write_slice(&self, ptr: u32, data: &[u8]) -> Result<(), wasmer::MemoryAccessError> {
        self.view().write(ptr as u64, data)
    }

    pub fn write_u32(&self, ptr: u32, value: u32) -> Result<(), wasmer::MemoryAccessError> {
        self.view().write(ptr as u64, &value.to_le_bytes())
    }
}

impl<E: EvmApi> MeteredMachine for HostioInfo<'_, E> {
    fn ink_left(&self) -> MachineMeter {
        let vm = self.env.meter();
        match vm.status() {
            0 => MachineMeter::Ready(vm.ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        // Write to WASM globals so middleware sees hostio gas charges.
        if let Some(ref g) = self.env.ink_global {
            let _ = g.set(&mut self.store, Value::I64(meter.ink().0 as i64));
        }
        if let Some(ref g) = self.env.ink_status_global {
            let _ = g.set(&mut self.store, Value::I32(meter.status() as i32));
        }
        // Also update MeterData.
        let vm = self.env.meter_mut();
        vm.set_ink(meter.ink());
        vm.set_status(meter.status());
    }

    // Override buy_ink to read current value from WASM globals first,
    // then deduct and write back to both globals and MeterData.
    fn buy_ink(&mut self, ink: Ink) -> Result<(), Escape> {
        // Read current ink from WASM globals (reflects middleware charges).
        let current = if let Some(ref g) = self.env.ink_global {
            if let Value::I64(v) = g.get(&mut self.store) {
                Ink(v as u64)
            } else {
                self.ink_ready()?
            }
        } else {
            self.ink_ready()?
        };
        if current < ink {
            self.set_meter(MachineMeter::Exhausted);
            return Escape::out_of_ink();
        }
        self.set_meter(MachineMeter::Ready(current - ink));
        Ok(())
    }

    fn require_ink(&mut self, ink: Ink) -> Result<(), Escape> {
        let current = if let Some(ref g) = self.env.ink_global {
            if let Value::I64(v) = g.get(&mut self.store) {
                Ink(v as u64)
            } else {
                self.ink_ready()?
            }
        } else {
            self.ink_ready()?
        };
        if current < ink {
            return Escape::out_of_ink();
        }
        Ok(())
    }
}

impl<E: EvmApi> GasMeteredMachine for HostioInfo<'_, E> {
    fn pricing(&self) -> crate::config::PricingParams {
        self.config().pricing
    }
}

impl<E: EvmApi> std::ops::Deref for HostioInfo<'_, E> {
    type Target = WasmEnv<E>;
    fn deref(&self) -> &Self::Target {
        self.env
    }
}

impl<E: EvmApi> std::ops::DerefMut for HostioInfo<'_, E> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.env
    }
}
