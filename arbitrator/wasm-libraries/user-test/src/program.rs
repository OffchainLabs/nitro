// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{ARGS, EVER_PAGES, EVM_DATA, KEYS, LOGS, OPEN_PAGES, OUTS};
use arbutil::{
    evm::{
        api::{EvmApi, VecReader},
        user::UserOutcomeKind,
        EvmData,
    },
    Bytes20, Bytes32, Color,
};
use caller_env::{static_caller::STATIC_MEM, GuestPtr, MemAccess};
use eyre::{eyre, Result};
use prover::programs::memory::MemoryModel;
use std::fmt::Display;
use user_host_trait::UserHost;

/// Signifies an out-of-bounds memory access was requested.
pub struct MemoryBoundsError;

impl From<MemoryBoundsError> for eyre::ErrReport {
    fn from(_: MemoryBoundsError) -> Self {
        eyre!("memory access out of bounds")
    }
}

/// Mock type representing a `user_host::Program`
pub struct Program {
    evm_api: MockEvmApi,
}

#[allow(clippy::unit_arg)]
impl UserHost<VecReader> for Program {
    type Err = eyre::ErrReport;
    type MemoryErr = MemoryBoundsError;
    type A = MockEvmApi;

    fn args(&self) -> &[u8] {
        unsafe { &ARGS }
    }

    fn outs(&mut self) -> &mut Vec<u8> {
        unsafe { &mut OUTS }
    }

    fn evm_api(&mut self) -> &mut Self::A {
        &mut self.evm_api
    }

    fn evm_data(&self) -> &EvmData {
        &EVM_DATA
    }

    fn evm_return_data_len(&mut self) -> &mut u32 {
        unimplemented!()
    }

    fn read_slice(&self, ptr: GuestPtr, len: u32) -> Result<Vec<u8>, MemoryBoundsError> {
        self.check_memory_access(ptr, len)?;
        unsafe { Ok(STATIC_MEM.read_slice(ptr, len as usize)) }
    }

    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> Result<[u8; N], MemoryBoundsError> {
        self.read_slice(ptr, N as u32)
            .map(|x| x.try_into().unwrap())
    }

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) -> Result<(), MemoryBoundsError> {
        self.check_memory_access(ptr, 4)?;
        unsafe { Ok(STATIC_MEM.write_u32(ptr, x)) }
    }

    fn write_slice(&self, ptr: GuestPtr, src: &[u8]) -> Result<(), MemoryBoundsError> {
        self.check_memory_access(ptr, src.len() as u32)?;
        unsafe { Ok(STATIC_MEM.write_slice(ptr, src)) }
    }

    fn say<D: Display>(&self, text: D) {
        println!("{} {text}", "Stylus says:".yellow());
    }

    fn trace(&mut self, name: &str, args: &[u8], outs: &[u8], _end_ink: u64) {
        let args = hex::encode(args);
        let outs = hex::encode(outs);
        println!("Error: unexpected hostio tracing info for {name} while proving: {args}, {outs}");
    }
}

impl Program {
    pub fn current() -> Self {
        Self {
            evm_api: MockEvmApi,
        }
    }

    fn check_memory_access(&self, _ptr: GuestPtr, _bytes: u32) -> Result<(), MemoryBoundsError> {
        Ok(()) // pretend we did a check
    }
}

pub struct MockEvmApi;

impl EvmApi<VecReader> for MockEvmApi {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let value = KEYS.lock().get(&key).cloned().unwrap_or_default();
        (value, 2100) // pretend worst case
    }

    fn cache_bytes32(&mut self, key: Bytes32, value: Bytes32) -> u64 {
        KEYS.lock().insert(key, value);
        0
    }

    fn flush_storage_cache(&mut self, _clear: bool, _gas_left: u64) -> Result<u64> {
        Ok(22100 * KEYS.lock().len() as u64) // pretend worst case
    }

    fn get_transient_bytes32(&mut self, _key: Bytes32) -> Bytes32 {
        unimplemented!()
    }

    fn set_transient_bytes32(&mut self, _key: Bytes32, _value: Bytes32) -> Result<()> {
        unimplemented!()
    }

    /// Simulates a contract call.
    /// Note: this call function is for testing purposes only and deviates from onchain behavior.
    fn contract_call(
        &mut self,
        _contract: Bytes20,
        _calldata: &[u8],
        _gas: u64,
        _value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        unimplemented!()
    }

    fn delegate_call(
        &mut self,
        _contract: Bytes20,
        _calldata: &[u8],
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        unimplemented!()
    }

    fn static_call(
        &mut self,
        _contract: Bytes20,
        _calldata: &[u8],
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        unimplemented!()
    }

    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!()
    }

    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _salt: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        unimplemented!()
    }

    fn get_return_data(&self) -> VecReader {
        unimplemented!()
    }

    fn emit_log(&mut self, data: Vec<u8>, _topics: u32) -> Result<()> {
        unsafe { LOGS.push(data) };
        Ok(())
    }

    fn account_balance(&mut self, _address: Bytes20) -> (Bytes32, u64) {
        unimplemented!()
    }

    fn account_code(&mut self, _address: Bytes20, _gas_left: u64) -> (VecReader, u64) {
        unimplemented!()
    }

    fn account_codehash(&mut self, _address: Bytes20) -> (Bytes32, u64) {
        unimplemented!()
    }

    fn add_pages(&mut self, pages: u16) -> u64 {
        let model = MemoryModel::new(2, 1000);
        unsafe {
            let (open, ever) = (OPEN_PAGES, EVER_PAGES);
            OPEN_PAGES = OPEN_PAGES.saturating_add(pages);
            EVER_PAGES = EVER_PAGES.max(OPEN_PAGES);
            model.gas_cost(pages, open, ever)
        }
    }

    fn capture_hostio(
        &mut self,
        _name: &str,
        _args: &[u8],
        _outs: &[u8],
        _start_ink: u64,
        _end_ink: u64,
    ) {
        unimplemented!()
    }
}
