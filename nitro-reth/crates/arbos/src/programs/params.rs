use alloy_primitives::B256;
use revm::Database;

use arb_storage::Storage;

/// ArbOS version constants for feature gating.
pub const ARBOS_VERSION_40: u64 = 40;
pub const ARBOS_VERSION_50: u64 = 50;
pub const ARBOS_VERSION_STYLUS_CONTRACT_LIMIT: u64 = 60;

// Initial parameter values.
const INITIAL_MAX_WASM_SIZE: u32 = 128 * 1024;
const INITIAL_STACK_DEPTH: u32 = 4 * 65536;
pub const INITIAL_FREE_PAGES: u16 = 2;
pub const INITIAL_PAGE_GAS: u16 = 1000;
pub const INITIAL_PAGE_RAMP: u64 = 620674314;
const INITIAL_PAGE_LIMIT: u16 = 128;
const INITIAL_INK_PRICE: u32 = 10000;
const INITIAL_MAX_FRAGMENT_COUNT: u8 = 2;
const INITIAL_MIN_INIT_GAS: u8 = 72;
const INITIAL_MIN_CACHED_GAS: u8 = 11;
const INITIAL_INIT_COST_SCALAR: u8 = 50;
const INITIAL_CACHED_COST_SCALAR: u8 = 50;
const INITIAL_EXPIRY_DAYS: u16 = 365;
const INITIAL_KEEPALIVE_DAYS: u16 = 31;
const INITIAL_RECENT_CACHE_SIZE: u16 = 32;

const V2_MIN_INIT_GAS: u8 = 69;

pub const MIN_CACHED_GAS_UNITS: u64 = 32;
pub const MIN_INIT_GAS_UNITS: u64 = 128;
pub const COST_SCALAR_PERCENT: u64 = 2;

const ARBOS_50_MAX_WASM_SIZE: u32 = 22000;

/// Stylus configuration parameters packed into storage words.
#[derive(Debug, Clone)]
pub struct StylusParams {
    pub arbos_version: u64,
    pub version: u16,
    pub ink_price: u32, // uint24 in Go, stored as u32
    pub max_stack_depth: u32,
    pub free_pages: u16,
    pub page_gas: u16,
    pub page_ramp: u64,
    pub page_limit: u16,
    pub min_init_gas: u8,
    pub min_cached_init_gas: u8,
    pub init_cost_scalar: u8,
    pub cached_cost_scalar: u8,
    pub expiry_days: u16,
    pub keepalive_days: u16,
    pub block_cache_size: u16,
    pub max_wasm_size: u32,
    pub max_fragment_count: u8,
}

impl StylusParams {
    /// Deserialize params from a storage substorage.
    pub fn load<D: Database>(arbos_version: u64, sto: &Storage<D>) -> Result<Self, ()> {
        let mut reader = PackedReader::new(sto);

        let mut params = StylusParams {
            arbos_version,
            version: reader.take_u16()?,
            ink_price: reader.take_u24()?,
            max_stack_depth: reader.take_u32()?,
            free_pages: reader.take_u16()?,
            page_gas: reader.take_u16()?,
            page_ramp: INITIAL_PAGE_RAMP,
            page_limit: reader.take_u16()?,
            min_init_gas: reader.take_u8()?,
            min_cached_init_gas: reader.take_u8()?,
            init_cost_scalar: reader.take_u8()?,
            cached_cost_scalar: reader.take_u8()?,
            expiry_days: reader.take_u16()?,
            keepalive_days: reader.take_u16()?,
            block_cache_size: reader.take_u16()?,
            max_wasm_size: 0,
            max_fragment_count: 0,
        };

        if arbos_version >= ARBOS_VERSION_40 {
            params.max_wasm_size = reader.take_u32()?;
        } else {
            params.max_wasm_size = INITIAL_MAX_WASM_SIZE;
        }
        if arbos_version >= ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
            params.max_fragment_count = reader.take_u8()?;
        }

        Ok(params)
    }

    /// Serialize and persist params to storage.
    pub fn save<D: Database>(&self, sto: &Storage<D>) -> Result<(), ()> {
        let mut data = Vec::with_capacity(32);

        data.extend_from_slice(&self.version.to_be_bytes());
        // uint24: 3 bytes big-endian
        data.push((self.ink_price >> 16) as u8);
        data.push((self.ink_price >> 8) as u8);
        data.push(self.ink_price as u8);
        data.extend_from_slice(&self.max_stack_depth.to_be_bytes());
        data.extend_from_slice(&self.free_pages.to_be_bytes());
        data.extend_from_slice(&self.page_gas.to_be_bytes());
        data.extend_from_slice(&self.page_limit.to_be_bytes());
        data.push(self.min_init_gas);
        data.push(self.min_cached_init_gas);
        data.push(self.init_cost_scalar);
        data.push(self.cached_cost_scalar);
        data.extend_from_slice(&self.expiry_days.to_be_bytes());
        data.extend_from_slice(&self.keepalive_days.to_be_bytes());
        data.extend_from_slice(&self.block_cache_size.to_be_bytes());

        if self.arbos_version >= ARBOS_VERSION_40 {
            data.extend_from_slice(&self.max_wasm_size.to_be_bytes());
        }
        if self.arbos_version >= ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
            data.push(self.max_fragment_count);
        }

        let mut slot = 0u64;
        let mut offset = 0;
        while offset < data.len() {
            let end = (offset + 32).min(data.len());
            let chunk = &data[offset..end];

            let mut word = [0u8; 32];
            word[..chunk.len()].copy_from_slice(chunk);
            sto.set_by_uint64(slot, B256::from(word))?;

            slot += 1;
            offset += 32;
        }
        Ok(())
    }

    /// Upgrade the params version (e.g. 1 -> 2).
    pub fn upgrade_to_version(&mut self, version: u16) -> Result<(), &'static str> {
        match version {
            2 => {
                if self.version != 1 {
                    return Err("unexpected version for upgrade to 2");
                }
                self.version = 2;
                self.min_init_gas = V2_MIN_INIT_GAS;
                Ok(())
            }
            _ => Err("unsupported version upgrade"),
        }
    }

    /// Handle ArbOS version upgrades that affect stylus params.
    pub fn upgrade_to_arbos_version(&mut self, new_arbos_version: u64) -> Result<(), &'static str> {
        if self.arbos_version >= new_arbos_version {
            return Err("unexpected arbos version downgrade");
        }

        match new_arbos_version {
            ARBOS_VERSION_50 => {
                if self.max_stack_depth > ARBOS_50_MAX_WASM_SIZE {
                    self.max_stack_depth = ARBOS_50_MAX_WASM_SIZE;
                }
            }
            ARBOS_VERSION_40 => {
                if self.version != 2 {
                    return Err("unexpected stylus version for arbos 40 upgrade");
                }
                self.max_wasm_size = INITIAL_MAX_WASM_SIZE;
            }
            ARBOS_VERSION_STYLUS_CONTRACT_LIMIT => {
                self.max_fragment_count = INITIAL_MAX_FRAGMENT_COUNT;
            }
            _ => {}
        }

        self.arbos_version = new_arbos_version;
        Ok(())
    }
}

/// Initialize default stylus params and persist to storage.
pub fn init_stylus_params<D: Database>(arbos_version: u64, sto: &Storage<D>) {
    let mut params = StylusParams {
        arbos_version,
        version: 1,
        ink_price: INITIAL_INK_PRICE,
        max_stack_depth: INITIAL_STACK_DEPTH,
        free_pages: INITIAL_FREE_PAGES,
        page_gas: INITIAL_PAGE_GAS,
        page_ramp: INITIAL_PAGE_RAMP,
        page_limit: INITIAL_PAGE_LIMIT,
        min_init_gas: INITIAL_MIN_INIT_GAS,
        min_cached_init_gas: INITIAL_MIN_CACHED_GAS,
        init_cost_scalar: INITIAL_INIT_COST_SCALAR,
        cached_cost_scalar: INITIAL_CACHED_COST_SCALAR,
        expiry_days: INITIAL_EXPIRY_DAYS,
        keepalive_days: INITIAL_KEEPALIVE_DAYS,
        block_cache_size: INITIAL_RECENT_CACHE_SIZE,
        max_wasm_size: 0,
        max_fragment_count: 0,
    };
    if arbos_version >= ARBOS_VERSION_40 {
        params.max_wasm_size = INITIAL_MAX_WASM_SIZE;
    }
    if arbos_version >= ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
        params.max_fragment_count = INITIAL_MAX_FRAGMENT_COUNT;
    }
    let _ = params.save(sto);
}

/// Helper to read packed fields from sequential storage words.
struct PackedReader<'a, D> {
    sto: &'a Storage<D>,
    slot: u64,
    buf: [u8; 32],
    pos: usize,
}

impl<'a, D: Database> PackedReader<'a, D> {
    fn new(sto: &'a Storage<D>) -> Self {
        Self {
            sto,
            slot: 0,
            buf: [0u8; 32],
            pos: 32, // force first read
        }
    }

    fn ensure(&mut self, count: usize) -> Result<(), ()> {
        if self.pos + count > 32 {
            let word = self.sto.get_by_uint64(self.slot)?;
            self.buf = word.0;
            self.slot += 1;
            self.pos = 0;
        }
        Ok(())
    }

    fn take_bytes(&mut self, count: usize) -> Result<&[u8], ()> {
        self.ensure(count)?;
        let start = self.pos;
        self.pos += count;
        Ok(&self.buf[start..self.pos])
    }

    fn take_u8(&mut self) -> Result<u8, ()> {
        let bytes = self.take_bytes(1)?;
        Ok(bytes[0])
    }

    fn take_u16(&mut self) -> Result<u16, ()> {
        let bytes = self.take_bytes(2)?;
        Ok(u16::from_be_bytes([bytes[0], bytes[1]]))
    }

    fn take_u24(&mut self) -> Result<u32, ()> {
        let bytes = self.take_bytes(3)?;
        Ok((bytes[0] as u32) << 16 | (bytes[1] as u32) << 8 | bytes[2] as u32)
    }

    fn take_u32(&mut self) -> Result<u32, ()> {
        let bytes = self.take_bytes(4)?;
        Ok(u32::from_be_bytes([bytes[0], bytes[1], bytes[2], bytes[3]]))
    }
}
