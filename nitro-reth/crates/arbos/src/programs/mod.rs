pub mod data_pricer;
pub mod memory;
pub mod params;

use alloy_primitives::B256;
use revm::Database;

use arb_storage::Storage;

use crate::address_set::{open_address_set, AddressSet};
use self::data_pricer::{init_data_pricer, open_data_pricer, DataPricer, ARBITRUM_START_TIME};
use self::memory::MemoryModel;
use self::params::{init_stylus_params, StylusParams};

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
    (a + b - 1) / b
}
