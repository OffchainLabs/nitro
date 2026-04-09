//! Shared test harness for Arbitrum precompile integration tests.
//!
//! Mirrors Nitro's `newMockEVMForTesting` from `precompiles/ArbAddressTable_test.go`:
//! a fresh in-memory revm context with configurable block/cfg/tx/state, against which a
//! `DynPrecompile` handler can be invoked through the real `EvmInternals` path.

#![allow(dead_code)]

use alloy_evm::{
    eth::EthEvmContext,
    precompiles::{DynPrecompile, Precompile, PrecompileInput},
    EvmInternals,
};
use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use revm::{
    database::{CacheDB, EmptyDB},
    precompile::{PrecompileError, PrecompileOutput, PrecompileResult},
    primitives::hardfork::SpecId,
    state::AccountInfo,
    Database,
};
use std::sync::{Mutex, MutexGuard, OnceLock};
use tiny_keccak::{Hasher, Keccak};

use arb_precompiles::storage_slot::{root_slot, ARBOS_STATE_ADDRESS, VERSION_OFFSET};

/// Process-wide lock that serialises precompile tests sharing global mutexes
/// (CURRENT_L2_BLOCK, L1_BLOCK_CACHE, L2_BLOCKHASH_CACHE, RECENT_WASMS, ...).
fn test_lock() -> MutexGuard<'static, ()> {
    static LOCK: OnceLock<Mutex<()>> = OnceLock::new();
    LOCK.get_or_init(|| Mutex::new(()))
        .lock()
        .unwrap_or_else(|p| p.into_inner())
}

/// 4-byte function selector helper. Mirrors keccak256(signature)[..4].
pub fn selector(sig: &str) -> [u8; 4] {
    let mut h = Keccak::v256();
    let mut out = [0u8; 32];
    h.update(sig.as_bytes());
    h.finalize(&mut out);
    [out[0], out[1], out[2], out[3]]
}

/// Build calldata: 4-byte selector + ABI-encoded 32-byte words.
pub fn calldata(sig: &str, args: &[B256]) -> Bytes {
    let mut buf = Vec::with_capacity(4 + args.len() * 32);
    buf.extend_from_slice(&selector(sig));
    for a in args {
        buf.extend_from_slice(a.as_slice());
    }
    Bytes::from(buf)
}

/// ABI-encode a U256 as a 32-byte word.
pub fn word_u256(v: U256) -> B256 {
    B256::from(v.to_be_bytes::<32>())
}

/// ABI-encode a u64 as a 32-byte word.
pub fn word_u64(v: u64) -> B256 {
    word_u256(U256::from(v))
}

/// ABI-encode an address as a 32-byte left-padded word.
pub fn word_address(a: Address) -> B256 {
    let mut out = [0u8; 32];
    out[12..].copy_from_slice(a.as_slice());
    B256::from(out)
}

/// Decode a single 32-byte word from precompile output.
pub fn decode_word(out: &Bytes, index: usize) -> B256 {
    let start = index * 32;
    let mut w = [0u8; 32];
    w.copy_from_slice(&out[start..start + 32]);
    B256::from(w)
}

/// Decode a U256 from precompile output (single word at offset 0).
pub fn decode_u256(out: &Bytes) -> U256 {
    U256::from_be_bytes(decode_word(out, 0).0)
}

/// Decode an address from precompile output (single word at offset 0).
pub fn decode_address(out: &Bytes) -> Address {
    let w = decode_word(out, 0);
    Address::from_slice(&w.0[12..])
}

/// Builder for a single precompile invocation.
pub struct PrecompileTest {
    db: CacheDB<EmptyDB>,
    spec: SpecId,
    block_number: u64,
    block_timestamp: u64,
    chain_id: u64,
    arbos_version: u64,
    caller: Address,
    target_address: Address,
    bytecode_address: Address,
    value: U256,
    is_static: bool,
    gas_limit: u64,
    evm_depth: usize,
    tx_is_aliased: bool,
}

impl Default for PrecompileTest {
    fn default() -> Self {
        Self {
            db: CacheDB::new(EmptyDB::default()),
            spec: SpecId::CANCUN,
            block_number: 1,
            block_timestamp: 1_700_000_000,
            chain_id: 42_161,
            arbos_version: 30,
            caller: Address::repeat_byte(0xAA),
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            gas_limit: 1_000_000,
            evm_depth: 1,
            tx_is_aliased: false,
        }
    }
}

impl PrecompileTest {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn spec(mut self, s: SpecId) -> Self {
        self.spec = s;
        self
    }
    pub fn arbos_version(mut self, v: u64) -> Self {
        self.arbos_version = v;
        self
    }
    pub fn chain_id(mut self, id: u64) -> Self {
        self.chain_id = id;
        self
    }
    pub fn caller(mut self, c: Address) -> Self {
        self.caller = c;
        self
    }
    pub fn target(mut self, t: Address) -> Self {
        self.target_address = t;
        self.bytecode_address = t;
        self
    }
    pub fn block(mut self, number: u64, timestamp: u64) -> Self {
        self.block_number = number;
        self.block_timestamp = timestamp;
        self
    }
    pub fn block_number(mut self, n: u64) -> Self {
        self.block_number = n;
        self
    }
    pub fn block_timestamp(mut self, ts: u64) -> Self {
        self.block_timestamp = ts;
        self
    }
    pub fn static_call(mut self, s: bool) -> Self {
        self.is_static = s;
        self
    }
    pub fn gas(mut self, g: u64) -> Self {
        self.gas_limit = g;
        self
    }
    pub fn value(mut self, v: U256) -> Self {
        self.value = v;
        self
    }
    pub fn evm_depth(mut self, d: usize) -> Self {
        self.evm_depth = d;
        self
    }
    pub fn tx_is_aliased(mut self, a: bool) -> Self {
        self.tx_is_aliased = a;
        self
    }

    /// Pre-populate an account.
    pub fn account(mut self, addr: Address, info: AccountInfo) -> Self {
        self.db.insert_account_info(addr, info);
        self
    }
    /// Pre-populate an account with the given balance and zero nonce/code.
    pub fn balance(mut self, addr: Address, balance: U256) -> Self {
        let info = self.db.basic(addr).ok().flatten().unwrap_or_default();
        self.db.insert_account_info(
            addr,
            AccountInfo {
                balance,
                nonce: info.nonce,
                code_hash: info.code_hash,
                code: info.code,
                ..Default::default()
            },
        );
        self
    }
    /// Pre-populate a single storage slot of an account. Implicitly creates the account.
    pub fn storage(mut self, addr: Address, slot: U256, value: U256) -> Self {
        if self.db.basic(addr).ok().flatten().is_none() {
            self.db.insert_account_info(addr, AccountInfo::default());
        }
        self.db
            .insert_account_storage(addr, slot, value)
            .expect("insert storage");
        self
    }

    /// Configure ArbOS state: ensures the ArbOS state account exists with nonce=1
    /// (so it survives EIP-161 deletion) and writes the raw arbos version into
    /// root slot 0. The raw value is what the protocol stores; the +55 offset
    /// applied by `arbOSVersion()` is purely a presentation concern.
    pub fn arbos_state(mut self) -> Self {
        let info = AccountInfo {
            nonce: 1,
            code_hash: keccak256([]),
            ..Default::default()
        };
        self.db.insert_account_info(ARBOS_STATE_ADDRESS, info);
        self.db
            .insert_account_storage(
                ARBOS_STATE_ADDRESS,
                root_slot(VERSION_OFFSET),
                U256::from(self.arbos_version),
            )
            .expect("insert arbos version");
        self
    }

    /// Run the precompile and return both the result and the post-state CacheDB
    /// for further assertion. Acquires a process-wide test lock for the duration of
    /// the call so the global state set via thread-locals/mutexes can't race with
    /// other tests.
    pub fn call(self, precompile: &DynPrecompile, input: &Bytes) -> PrecompileRun {
        let _guard = test_lock();

        arb_precompiles::set_arbos_version(self.arbos_version);
        arb_precompiles::set_l1_block_number_for_evm(self.block_number);
        arb_precompiles::set_block_timestamp(self.block_timestamp);
        arb_precompiles::set_evm_depth(self.evm_depth);
        arb_precompiles::set_current_l2_block(self.block_number);
        arb_precompiles::set_tx_is_aliased(self.tx_is_aliased);

        let mut ctx = EthEvmContext::new(self.db, self.spec);
        ctx.cfg.chain_id = self.chain_id;
        ctx.block.number = U256::from(self.block_number);
        ctx.block.timestamp = U256::from(self.block_timestamp);
        ctx.tx.caller = self.caller;

        let result = {
            let internals = EvmInternals::from_context(&mut ctx);
            precompile.call(PrecompileInput {
                data: input,
                gas: self.gas_limit,
                caller: self.caller,
                value: self.value,
                is_static: self.is_static,
                internals,
                target_address: self.target_address,
                bytecode_address: self.bytecode_address,
            })
        };

        PrecompileRun {
            result,
            db: ctx.journaled_state.database,
        }
    }
}

/// Outcome of a single precompile call. Owns the post-state DB.
pub struct PrecompileRun {
    pub result: PrecompileResult,
    pub db: CacheDB<EmptyDB>,
}

impl PrecompileRun {
    /// Unwrap to a successful PrecompileOutput, panicking with the error otherwise.
    pub fn assert_ok(&self) -> &PrecompileOutput {
        match &self.result {
            Ok(out) => out,
            Err(e) => panic!("expected Ok, got Err: {e:?}"),
        }
    }
    /// Assert the precompile errored.
    pub fn assert_err(&self) -> &PrecompileError {
        match &self.result {
            Err(e) => e,
            Ok(out) => panic!("expected Err, got Ok with {} bytes", out.bytes.len()),
        }
    }
    pub fn output(&self) -> &Bytes {
        &self.assert_ok().bytes
    }
    pub fn gas_used(&self) -> u64 {
        self.assert_ok().gas_used
    }
    pub fn storage(&self, addr: Address, slot: U256) -> U256 {
        self.db
            .cache.accounts
            .get(&addr)
            .and_then(|a| a.storage.get(&slot).copied())
            .unwrap_or(U256::ZERO)
    }
    pub fn balance(&self, addr: Address) -> U256 {
        self.db
            .cache.accounts
            .get(&addr)
            .map(|a| a.info.balance)
            .unwrap_or(U256::ZERO)
    }
}
