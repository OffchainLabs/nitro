pub mod api;
pub mod data_pricer;
pub mod memory;
pub mod params;
pub mod types;

use alloy_primitives::B256;
use arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_FIXES;
use arb_primitives::multigas::{MultiGas, ResourceKind};
use revm::Database;

use arb_storage::Storage;

pub use self::types::{
    evm_memory_cost, to_word_size, ActivationResult, EvmData, ProgParams, RequestType, UserOutcome,
};
use self::{
    data_pricer::{init_data_pricer, open_data_pricer, DataPricer, ARBITRUM_START_TIME},
    memory::MemoryModel,
    params::{init_stylus_params, StylusParams},
};
use crate::address_set::{open_address_set, AddressSet};

const PARAMS_KEY: &[u8] = &[0];
const PROGRAM_DATA_KEY: &[u8] = &[1];
const MODULE_HASHES_KEY: &[u8] = &[2];
const DATA_PRICER_KEY: &[u8] = &[3];
const CACHE_MANAGERS_KEY: &[u8] = &[4];

/// Per-program metadata stored in state.
#[derive(Debug, Clone, Copy)]
pub struct Program {
    pub version: u16,
    pub init_cost: u16,
    pub cached_cost: u16,
    pub footprint: u16,
    pub asm_estimate_kb: u32, // uint24 in Go
    pub activated_at: u32,    // uint24 hours since Arbitrum began
    pub age_seconds: u64,     // not stored in state
    pub cached: bool,
}

impl Program {
    /// Decode a program from a 32-byte storage word.
    pub fn from_storage(data: B256, time: u64) -> Self {
        let b = data.as_slice();
        let version = u16::from_be_bytes([b[0], b[1]]);
        let init_cost = u16::from_be_bytes([b[2], b[3]]);
        let cached_cost = u16::from_be_bytes([b[4], b[5]]);
        let footprint = u16::from_be_bytes([b[6], b[7]]);
        let activated_at = (b[8] as u32) << 16 | (b[9] as u32) << 8 | b[10] as u32;
        let asm_estimate_kb = (b[11] as u32) << 16 | (b[12] as u32) << 8 | b[13] as u32;
        let cached = b[14] != 0;
        let age_seconds = hours_to_age(time, activated_at);
        Program {
            version,
            init_cost,
            cached_cost,
            footprint,
            asm_estimate_kb,
            activated_at,
            age_seconds,
            cached,
        }
    }

    /// Encode the program to a 32-byte storage word.
    pub fn to_storage(&self) -> B256 {
        let mut data = [0u8; 32];
        data[0..2].copy_from_slice(&self.version.to_be_bytes());
        data[2..4].copy_from_slice(&self.init_cost.to_be_bytes());
        data[4..6].copy_from_slice(&self.cached_cost.to_be_bytes());
        data[6..8].copy_from_slice(&self.footprint.to_be_bytes());
        // activated_at: uint24
        data[8] = (self.activated_at >> 16) as u8;
        data[9] = (self.activated_at >> 8) as u8;
        data[10] = self.activated_at as u8;
        // asm_estimate_kb: uint24
        data[11] = (self.asm_estimate_kb >> 16) as u8;
        data[12] = (self.asm_estimate_kb >> 8) as u8;
        data[13] = self.asm_estimate_kb as u8;
        data[14] = self.cached as u8;
        B256::from(data)
    }

    /// Estimated ASM size in bytes.
    pub fn asm_size(&self) -> u32 {
        self.asm_estimate_kb.saturating_mul(1024)
    }

    /// Gas cost for program initialization.
    pub fn init_gas(&self, params: &StylusParams) -> u64 {
        let base = (params.min_init_gas as u64).saturating_mul(params::MIN_INIT_GAS_UNITS);
        let dyno = (self.init_cost as u64)
            .saturating_mul((params.init_cost_scalar as u64) * params::COST_SCALAR_PERCENT);
        base.saturating_add(div_ceil(dyno, 100))
    }

    /// Gas cost for cached program initialization.
    pub fn cached_gas(&self, params: &StylusParams) -> u64 {
        let base = (params.min_cached_init_gas as u64).saturating_mul(params::MIN_CACHED_GAS_UNITS);
        let dyno = (self.cached_cost as u64)
            .saturating_mul((params.cached_cost_scalar as u64) * params::COST_SCALAR_PERCENT);
        base.saturating_add(div_ceil(dyno, 100))
    }
}

/// Stylus programs state.
pub struct Programs<D> {
    pub arbos_version: u64,
    pub backing_storage: Storage<D>,
    programs: Storage<D>,
    module_hashes: Storage<D>,
    pub data_pricer: DataPricer<D>,
    pub cache_managers: AddressSet<D>,
}

impl<D: Database> Programs<D> {
    pub fn initialize(arbos_version: u64, sto: &Storage<D>) {
        let params_sto = sto.open_sub_storage(PARAMS_KEY);
        init_stylus_params(arbos_version, &params_sto);
        let data_pricer_sto = sto.open_sub_storage(DATA_PRICER_KEY);
        init_data_pricer(&data_pricer_sto);
    }

    pub fn open(arbos_version: u64, sto: Storage<D>) -> Self {
        let data_pricer_sto = sto.open_sub_storage(DATA_PRICER_KEY);
        let data_pricer = open_data_pricer(&data_pricer_sto);
        let programs = sto.open_sub_storage(PROGRAM_DATA_KEY);
        let module_hashes = sto.open_sub_storage(MODULE_HASHES_KEY);
        let cache_managers_sto = sto.open_sub_storage(CACHE_MANAGERS_KEY);
        let cache_managers = open_address_set(cache_managers_sto);
        Self {
            arbos_version,
            backing_storage: sto,
            programs,
            module_hashes,
            data_pricer,
            cache_managers,
        }
    }

    /// Load the current Stylus parameters.
    pub fn params(&self) -> Result<StylusParams, ()> {
        let sto = self.backing_storage.open_sub_storage(PARAMS_KEY);
        StylusParams::load(self.arbos_version, &sto)
    }

    /// Retrieve a program entry (may be expired or unactivated).
    pub fn get_program(&self, code_hash: B256, time: u64) -> Result<Program, ()> {
        let data = self.programs.get(code_hash)?;
        Ok(Program::from_storage(data, time))
    }

    /// Retrieve and validate an active program.
    pub fn get_active_program(
        &self,
        code_hash: B256,
        time: u64,
        params: &StylusParams,
    ) -> Result<Program, ()> {
        let program = self.get_program(code_hash, time)?;
        if program.version == 0 {
            return Err(());
        }
        if program.version != params.version {
            return Err(());
        }
        if program.age_seconds > days_to_seconds(params.expiry_days) {
            return Err(());
        }
        Ok(program)
    }

    /// Store a program entry.
    pub fn set_program(&self, code_hash: B256, program: Program) -> Result<(), ()> {
        self.programs.set(code_hash, program.to_storage())
    }

    /// Check if a program exists and its status.
    pub fn program_exists(
        &self,
        code_hash: B256,
        time: u64,
        params: &StylusParams,
    ) -> Result<(u16, bool, bool), ()> {
        let program = self.get_program(code_hash, time)?;
        let expired = program.activated_at == 0
            || hours_to_age(time, program.activated_at) > days_to_seconds(params.expiry_days);
        Ok((program.version, expired, program.cached))
    }

    /// Get the module hash for a code hash.
    pub fn get_module_hash(&self, code_hash: B256) -> Result<B256, ()> {
        self.module_hashes.get(code_hash)
    }

    /// Set the module hash for a code hash.
    pub fn set_module_hash(&self, code_hash: B256, module_hash: B256) -> Result<(), ()> {
        self.module_hashes.set(code_hash, module_hash)
    }

    /// Build runtime parameters for a program invocation.
    pub fn prog_params(&self, version: u16, debug_mode: bool, params: &StylusParams) -> ProgParams {
        ProgParams {
            version,
            max_depth: params.max_stack_depth,
            ink_price: params.ink_price,
            debug_mode,
        }
    }

    /// Activate a Stylus program. Records metadata and charges data fees.
    ///
    /// Returns `(version, code_hash, module_hash, data_fee)` on success.
    pub fn activate_program(
        &self,
        code_hash: B256,
        wasm: &[u8],
        time: u64,
        page_limit: u16,
        debug: bool,
        activate_fn: impl FnOnce(&[u8], u16, u64, u16, bool) -> Result<ActivationResult, String>,
    ) -> Result<(u16, B256, alloy_primitives::U256), String> {
        let params = self.params().map_err(|_| "failed to load params")?;
        let stylus_version = params.version;

        let (current_version, expired, cached) = self
            .program_exists(code_hash, time, &params)
            .map_err(|_| "failed to read program")?;

        if current_version == stylus_version && !expired {
            return Err("program up to date".into());
        }

        let info = activate_fn(wasm, stylus_version, self.arbos_version, page_limit, debug)?;

        // If previously cached, remove old module.
        if cached {
            // Old module eviction would happen at the runtime layer.
        }

        self.set_module_hash(code_hash, info.module_hash)
            .map_err(|_| "failed to set module hash")?;

        let estimate_kb = div_ceil(info.asm_estimate as u64, 1024) as u32;

        let data_fee = self
            .data_pricer
            .update_model(info.asm_estimate, time)
            .map_err(|_| "failed to update data pricer")?;

        let program = Program {
            version: stylus_version,
            init_cost: info.init_gas,
            cached_cost: info.cached_init_gas,
            footprint: info.footprint,
            asm_estimate_kb: estimate_kb.min(0xFF_FFFF), // uint24 max
            activated_at: hours_since_arbitrum(time),
            age_seconds: 0,
            cached,
        };

        self.set_program(code_hash, program)
            .map_err(|_| "failed to set program")?;

        Ok((stylus_version, info.module_hash, data_fee))
    }

    /// Compute gas costs for calling a Stylus program.
    ///
    /// Returns `(call_gas_cost, memory_model)`.
    pub fn call_gas_cost(
        &self,
        code_hash: B256,
        time: u64,
        pages_open: u16,
        recent_cache_hit: bool,
    ) -> Result<(u64, Program, MemoryModel), ()> {
        let params = self.params()?;
        let program = self.get_active_program(code_hash, time, &params)?;
        let model = MemoryModel::new(params.free_pages, params.page_gas);

        let mut cost = model.gas_cost(program.footprint, pages_open, pages_open);

        let cached = program.cached || recent_cache_hit;
        if cached || program.version > 1 {
            cost = cost.saturating_add(program.cached_gas(&params));
        }
        if !cached {
            cost = cost.saturating_add(program.init_gas(&params));
        }

        Ok((cost, program, model))
    }

    /// Extend a program's expiry by resetting its activation time.
    ///
    /// Returns the data fee charged.
    pub fn program_keepalive(
        &self,
        code_hash: B256,
        time: u64,
    ) -> Result<alloy_primitives::U256, String> {
        let params = self.params().map_err(|_| "failed to load params")?;
        let mut program = self
            .get_active_program(code_hash, time, &params)
            .map_err(|_| "program not active")?;

        if program.age_seconds < days_to_seconds(params.keepalive_days) {
            return Err("keepalive too soon".into());
        }
        if program.version != params.version {
            return Err("program needs upgrade".into());
        }

        let data_fee = self
            .data_pricer
            .update_model(program.asm_size(), time)
            .map_err(|_| "failed to update data pricer")?;

        program.activated_at = hours_since_arbitrum(time);
        self.set_program(code_hash, program)
            .map_err(|_| "failed to set program")?;

        Ok(data_fee)
    }

    /// Execute a Stylus program call.
    ///
    /// Computes gas costs, resolves program metadata, and delegates to the
    /// provided `call_fn` which handles the actual WASM runtime execution.
    /// After execution, enforces return data cost parity with the EVM and
    /// attributes residual gas to WasmComputation.
    ///
    /// The `call_fn` receives `(program, prog_params, evm_data, calldata, gas)`
    /// and returns `(output_bytes, gas_left)` or an error.
    pub fn call_program<F>(
        &self,
        code_hash: B256,
        time: u64,
        pages_open: u16,
        calldata: &[u8],
        evm_data: EvmData,
        _reentrant: bool,
        debug_mode: bool,
        gas: u64,
        used_multi_gas: &mut MultiGas,
        recent_cache_hit: bool,
        call_fn: F,
    ) -> Result<(Vec<u8>, u64), String>
    where
        F: FnOnce(Program, ProgParams, EvmData, &[u8], u64) -> Result<(Vec<u8>, u64), String>,
    {
        let (call_cost, program, _model) = self
            .call_gas_cost(code_hash, time, pages_open, recent_cache_hit)
            .map_err(|_| "failed to compute call cost")?;

        if gas < call_cost {
            return Err("insufficient gas for Stylus call".into());
        }

        let gas_for_program = gas - call_cost;
        let params = self.prog_params(
            program.version,
            debug_mode,
            &self.params().map_err(|_| "failed to load params")?,
        );

        let starting_gas = gas;
        let (output, gas_left) = call_fn(program, params, evm_data, calldata, gas_for_program)?;

        // Ensure return data costs at least as much as it would in the EVM.
        let gas_left = if !output.is_empty() && self.arbos_version >= ARBOS_VERSION_STYLUS_FIXES {
            let evm_cost = evm_memory_cost(output.len() as u64);
            if starting_gas < evm_cost {
                // Burn all remaining gas.
                attribute_wasm_computation(used_multi_gas, starting_gas, 0);
                return Err("out of gas".into());
            }
            let max_gas_to_return = starting_gas - evm_cost;
            gas_left.min(max_gas_to_return)
        } else {
            gas_left
        };

        attribute_wasm_computation(used_multi_gas, starting_gas, gas_left);

        Ok((output, gas_left))
    }

    /// Update the cached status of a program.
    pub fn set_program_cached(
        &self,
        code_hash: B256,
        cache: bool,
        time: u64,
    ) -> Result<(), String> {
        let params = self.params().map_err(|_| "failed to load params")?;
        let mut program = self
            .get_program(code_hash, time)
            .map_err(|_| "failed to read program")?;

        let expired = program.age_seconds > days_to_seconds(params.expiry_days);

        if program.version != params.version && cache {
            return Err("program needs upgrade".into());
        }
        if expired && cache {
            return Err("program expired".into());
        }
        if program.cached == cache {
            return Ok(());
        }

        program.cached = cache;
        self.set_program(code_hash, program)
            .map_err(|_| "failed to set program")?;

        Ok(())
    }
}

/// Information returned from program activation.
#[derive(Debug, Clone)]
pub struct ActivationInfo {
    pub module_hash: B256,
    pub init_gas: u16,
    pub cached_init_gas: u16,
    pub asm_estimate: u32,
    pub footprint: u16,
}

/// Attribute residual WASM gas consumption to the WasmComputation resource kind.
///
/// After a Stylus program executes, the total gas consumed may exceed what was
/// individually tracked through MultiGas accounting. This function assigns the
/// residual (unaccounted) gas to `ResourceKind::WasmComputation`.
pub fn attribute_wasm_computation(used_multi_gas: &mut MultiGas, starting_gas: u64, gas_left: u64) {
    let used_gas = starting_gas.saturating_sub(gas_left);
    let accounted_gas = used_multi_gas.single_gas();

    let residual = if accounted_gas > used_gas {
        tracing::trace!(
            used_gas,
            accounted_gas,
            "negative WASM computation residual"
        );
        0
    } else {
        used_gas - accounted_gas
    };

    let (updated, overflow) =
        used_multi_gas.safe_increment(ResourceKind::WasmComputation, residual);
    if overflow {
        tracing::trace!(residual, "WASM computation gas overflow");
    }
    *used_multi_gas = updated;
}

/// Hours since Arbitrum began, rounded down.
pub fn hours_since_arbitrum(time: u64) -> u32 {
    let elapsed = time.saturating_sub(ARBITRUM_START_TIME);
    (elapsed / 3600).min(u32::MAX as u64) as u32
}

/// Compute program age in seconds from hours since Arbitrum began.
pub fn hours_to_age(time: u64, hours: u32) -> u64 {
    let seconds = (hours as u64).saturating_mul(3600);
    let activated_at = ARBITRUM_START_TIME.saturating_add(seconds);
    time.saturating_sub(activated_at)
}

fn days_to_seconds(days: u16) -> u64 {
    (days as u64) * 24 * 3600
}

fn div_ceil(a: u64, b: u64) -> u64 {
    a.div_ceil(b)
}
