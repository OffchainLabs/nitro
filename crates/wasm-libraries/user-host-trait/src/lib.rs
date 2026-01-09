// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::{
    benchmark::Benchmark,
    crypto,
    evm::{
        self,
        api::{CreateRespone, DataReader, EvmApi, Gas, Ink},
        storage::StorageCache,
        user::UserOutcomeKind,
        EvmData, ARBOS_VERSION_STYLUS_CHARGING_FIXES,
    },
    pricing::{hostio, EVM_API_INK},
    Bytes20, Bytes32,
};
pub use caller_env::GuestPtr;
use eyre::{eyre, Result};
use prover::{
    programs::{meter::OutOfInkError, prelude::*},
    value::Value,
};
use ruint2::Uint;
use std::fmt::Display;
use std::time::Instant;

macro_rules! be {
    ($int:expr) => {
        $int.to_be_bytes()
    };
}

macro_rules! trace {
    ($name:expr, $env:expr, [$($args:expr),+], [$($outs:expr),+], $ret:expr) => {{
        if $env.evm_data().tracing {
            let end_ink = $env.ink_ready()?;
            let mut args = vec![];
            $(args.extend($args);)*
            let mut outs = vec![];
            $(outs.extend($outs);)*
            $env.trace($name, &args, &outs, end_ink);
        }
        Ok($ret)
    }};
    ($name:expr, $env:expr, [$($args:expr),+], $outs:expr) => {{
        trace!($name, $env, [$($args),+], $outs, ())
    }};
    ($name:expr, $env:expr, $args:expr, $outs:expr) => {{
        trace!($name, $env, $args, $outs, ())
    }};
    ($name:expr, $env:expr, [$($args:expr),+], $outs:expr, $ret:expr) => {
        trace!($name, $env, [$($args),+], [$outs], $ret)
    };
    ($name:expr, $env:expr, $args:expr, $outs:expr, $ret:expr) => {
        trace!($name, $env, [$args], [$outs], $ret)
    };
}
type Address = Bytes20;
type Wei = Bytes32;
type U256 = Uint<256, 4>;

pub struct CallInputs {
    pub contract: Address,
    pub input: Vec<u8>,
    pub gas_left: Gas,
    pub gas_req: Gas,
    pub value: Option<Wei>,
}

#[allow(clippy::too_many_arguments)]
pub trait UserHost<DR: DataReader>: GasMeteredMachine {
    type Err: From<OutOfInkError> + From<Self::MemoryErr> + From<eyre::ErrReport>;
    type MemoryErr;
    type A: EvmApi<DR>;

    fn args(&self) -> &[u8];
    fn outs(&mut self) -> &mut Vec<u8>;

    fn evm_api(&mut self) -> &mut Self::A;
    fn evm_data(&self) -> &EvmData;
    fn benchmark(&mut self) -> &mut Benchmark;
    fn evm_return_data_len(&mut self) -> &mut u32;

    fn read_slice(&self, ptr: GuestPtr, len: u32) -> Result<Vec<u8>, Self::MemoryErr>;
    fn read_fixed<const N: usize>(&self, ptr: GuestPtr) -> Result<[u8; N], Self::MemoryErr>;

    fn write_u32(&mut self, ptr: GuestPtr, x: u32) -> Result<(), Self::MemoryErr>;
    fn write_slice(&self, ptr: GuestPtr, src: &[u8]) -> Result<(), Self::MemoryErr>;

    fn read_bytes20(&self, ptr: GuestPtr) -> Result<Bytes20, Self::MemoryErr> {
        self.read_fixed(ptr).map(Into::into)
    }
    fn read_bytes32(&self, ptr: GuestPtr) -> Result<Bytes32, Self::MemoryErr> {
        self.read_fixed(ptr).map(Into::into)
    }
    fn read_u256(&self, ptr: GuestPtr) -> Result<(U256, Bytes32), Self::MemoryErr> {
        let value = self.read_bytes32(ptr)?;
        Ok((value.into(), value))
    }

    fn say<D: Display>(&self, text: D);
    fn trace(&mut self, name: &str, args: &[u8], outs: &[u8], end_ink: Ink);

    fn write_bytes20(&self, ptr: GuestPtr, src: Bytes20) -> Result<(), Self::MemoryErr> {
        self.write_slice(ptr, &src.0)
    }
    fn write_bytes32(&self, ptr: GuestPtr, src: Bytes32) -> Result<(), Self::MemoryErr> {
        self.write_slice(ptr, &src.0)
    }

    /// Parses guest pointer types into more structured Rust types to use
    /// across methods that perform contract calls for better developer ergonomics.
    fn parse_call_inputs(
        &mut self,
        contract: GuestPtr,
        data: GuestPtr,
        gas: Gas,
        data_len: u32,
        value: Option<GuestPtr>,
    ) -> Result<CallInputs, Self::Err> {
        let gas_left = self.gas_left()?;
        let gas_req = gas.min(gas_left);
        let contract = self.read_bytes20(contract)?;
        let input = self.read_slice(data, data_len)?;
        let value = value.map(|x| self.read_bytes32(x)).transpose()?;
        Ok(CallInputs {
            contract,
            input,
            gas_left,
            gas_req,
            value,
        })
    }

    /// Reads the program calldata. The semantics are equivalent to that of the EVM's
    /// [`CALLDATA_COPY`] opcode when requesting the entirety of the current call's calldata.
    ///
    /// [`CALLDATA_COPY`]: https://www.evm.codes/#37
    fn read_args(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::READ_ARGS_BASE_INK)?;
        self.pay_for_write(self.args().len() as u32)?;
        self.write_slice(ptr, self.args())?;
        trace!("read_args", self, &[], self.args())
    }

    /// Writes the final return data. If not called before the program exists, the return data will
    /// be 0 bytes long. Note that this hostio does not cause the program to exit, which happens
    /// naturally when `user_entrypoint` returns.
    fn write_result(&mut self, ptr: GuestPtr, len: u32) -> Result<(), Self::Err> {
        self.buy_ink(hostio::WRITE_RESULT_BASE_INK)?;
        self.pay_for_read(len)?;
        self.pay_for_read(len)?; // read from geth
        *self.outs() = self.read_slice(ptr, len)?;
        trace!("write_result", self, &*self.outs(), &[])
    }

    /// Exits program execution early with the given status code.
    /// If `0`, the program returns successfully with any data supplied by `write_result`.
    /// Otherwise, the program reverts and treats any `write_result` data as revert data.
    ///
    /// The semantics are equivalent to that of the EVM's [`Return`] and [`Revert`] opcodes.
    /// Note: this function just traces, it's up to the caller to actually perform the exit.
    ///
    /// [`Return`]: https://www.evm.codes/#f3
    /// [`Revert`]: https://www.evm.codes/#fd
    fn exit_early(&mut self, status: u32) -> Result<(), Self::Err> {
        trace!("exit_early", self, be!(status), &[])
    }

    /// Reads a 32-byte value from permanent storage. Stylus's storage format is identical to
    /// that of the EVM. This means that, under the hood, this hostio is accessing the 32-byte
    /// value stored in the EVM state trie at offset `key`, which will be `0` when not previously
    /// set. The semantics, then, are equivalent to that of the EVM's [`SLOAD`] opcode.
    ///
    /// Note: the Stylus VM implements storage caching. This means that repeated calls to the same key
    /// will cost less than in the EVM.
    ///
    /// [`SLOAD`]: https://www.evm.codes/#54
    fn storage_load_bytes32(&mut self, key: GuestPtr, dest: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::STORAGE_LOAD_BASE_INK)?;
        let arbos_version = self.evm_data().arbos_version;

        // require for cache-miss case, preserve wrong behavior for old arbos
        let evm_api_gas_to_use = if arbos_version < ARBOS_VERSION_STYLUS_CHARGING_FIXES {
            Gas(EVM_API_INK.0)
        } else {
            self.pricing().ink_to_gas(EVM_API_INK)
        };
        self.require_gas(
            evm::COLD_SLOAD_GAS + StorageCache::REQUIRED_ACCESS_GAS + evm_api_gas_to_use,
        )?;
        let key = self.read_bytes32(key)?;

        let (value, gas_cost) = self.evm_api().get_bytes32(key, evm_api_gas_to_use)?;
        self.buy_gas(gas_cost)?;
        self.write_bytes32(dest, value)?;
        trace!("storage_load_bytes32", self, key, value)
    }

    /// Writes a 32-byte value to the permanent storage cache. Stylus's storage format is identical to that
    /// of the EVM. This means that, under the hood, this hostio represents storing a 32-byte value into
    /// the EVM state trie at offset `key`. Refunds are tabulated exactly as in the EVM. The semantics, then,
    /// are equivalent to that of the EVM's [`SSTORE`] opcode.
    ///
    /// Note: because this value is cached, one must call `storage_flush_cache` to persist the value.
    ///
    /// Auditor's note: we require the [`SSTORE`] sentry per EVM rules. The `gas_cost` returned by the EVM API
    /// may exceed this amount, but that's ok because the predominant cost is due to state bloat concerns.
    ///
    /// [`SSTORE`]: https://www.evm.codes/#55
    fn storage_cache_bytes32(&mut self, key: GuestPtr, value: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::STORAGE_CACHE_BASE_INK)?;
        self.require_gas(evm::SSTORE_SENTRY_GAS + StorageCache::REQUIRED_ACCESS_GAS)?; // see operations_acl_arbitrum.go

        let key = self.read_bytes32(key)?;
        let value = self.read_bytes32(value)?;

        let gas_cost = self.evm_api().cache_bytes32(key, value)?;
        self.buy_gas(gas_cost)?;
        trace!("storage_cache_bytes32", self, [key, value], &[])
    }

    /// Persists any dirty values in the storage cache to the EVM state trie, dropping the cache entirely if requested.
    /// Analogous to repeated invocations of [`SSTORE`].
    ///
    /// [`SSTORE`]: https://www.evm.codes/#55
    fn storage_flush_cache(&mut self, clear: bool) -> Result<(), Self::Err> {
        self.buy_ink(hostio::STORAGE_FLUSH_BASE_INK)?;
        self.require_gas(evm::SSTORE_SENTRY_GAS)?; // see operations_acl_arbitrum.go

        let gas_left = self.gas_left()?;
        let (gas_cost, outcome) = self
            .evm_api()
            .flush_storage_cache(clear, gas_left + Gas(1))?;
        if self.evm_data().arbos_version >= ARBOS_VERSION_STYLUS_CHARGING_FIXES {
            self.buy_gas(gas_cost)?;
        }
        if outcome != UserOutcomeKind::Success {
            return Err(eyre!("outcome {outcome:?}").into());
        }
        trace!("storage_flush_cache", self, [be!(clear as u8)], &[])
    }

    /// Reads a 32-byte value from transient storage. Stylus's storage format is identical to
    /// that of the EVM. This means that, under the hood, this hostio is accessing the 32-byte
    /// value stored in the EVM's transient state trie at offset `key`, which will be `0` when not previously
    /// set. The semantics, then, are equivalent to that of the EVM's [`TLOAD`] opcode.
    ///
    /// [`TLOAD`]: https://www.evm.codes/#5c
    fn transient_load_bytes32(&mut self, key: GuestPtr, dest: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::TRANSIENT_LOAD_BASE_INK)?;
        self.buy_gas(evm::TLOAD_GAS)?;

        let key = self.read_bytes32(key)?;
        let value = self.evm_api().get_transient_bytes32(key)?;
        self.write_bytes32(dest, value)?;
        trace!("transient_load_bytes32", self, key, value)
    }

    /// Writes a 32-byte value to transient storage. Stylus's storage format is identical to that
    /// of the EVM. This means that, under the hood, this hostio represents storing a 32-byte value into
    /// the EVM's transient state trie at offset `key`. The semantics, then, are equivalent to that of the
    /// EVM's [`TSTORE`] opcode.
    ///
    /// [`TSTORE`]: https://www.evm.codes/#5d
    fn transient_store_bytes32(&mut self, key: GuestPtr, value: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::TRANSIENT_STORE_BASE_INK)?;
        self.buy_gas(evm::TSTORE_GAS)?;

        let key = self.read_bytes32(key)?;
        let value = self.read_bytes32(value)?;
        let outcome = self.evm_api().set_transient_bytes32(key, value)?;
        if outcome != UserOutcomeKind::Success {
            return Err(eyre!("outcome {outcome:?}").into());
        }
        trace!("transient_store_bytes32", self, [key, value], &[])
    }

    /// Calls the contract at the given address with options for passing value and to limit the
    /// amount of gas supplied. The return status indicates whether the call succeeded, and is
    /// nonzero on failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which can
    /// be read via the `read_return_data` hostio. The bytes are not returned directly so that the
    /// programmer can potentially save gas by choosing which subset of the return result they'd
    /// like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`CALL`] opcode, including callvalue
    /// stipends and the 63/64 gas rule. This means that supplying the `u64::MAX` gas can be used
    /// to send as much as possible.
    ///
    /// [`CALL`]: https://www.evm.codes/#f1
    fn call_contract(
        &mut self,
        contract: GuestPtr,
        data: GuestPtr,
        data_len: u32,
        value: GuestPtr,
        gas: Gas,
        ret_len: GuestPtr,
    ) -> Result<u8, Self::Err> {
        self.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
        self.pay_for_read(data_len)?;
        self.pay_for_read(data_len)?; // read from geth

        let CallInputs {
            contract,
            input,
            gas_left,
            gas_req,
            value,
        } = self.parse_call_inputs(contract, data, gas, data_len, Some(value))?;

        let (outs_len, gas_cost, status) =
            self.evm_api()
                .contract_call(contract, &input, gas_left, gas_req, value.unwrap())?;

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = outs_len;
        self.write_u32(ret_len, outs_len)?;
        let status = status as u8;

        if self.evm_data().tracing {
            let value = value.into_iter().flatten();
            return trace!(
                "call_contract",
                self,
                [contract, be!(gas), value, &input],
                [be!(outs_len), be!(status)],
                status
            );
        }
        Ok(status)
    }

    /// Delegate calls the contract at the given address, with the option to limit the amount of
    /// gas supplied. The return status indicates whether the call succeeded, and is nonzero on
    /// failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which
    /// can be read via the `read_return_data` hostio. The bytes are not returned directly so that
    /// the programmer can potentially save gas by choosing which subset of the return result
    /// they'd like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`DELEGATE_CALL`] opcode, including the
    /// 63/64 gas rule. This means that supplying `u64::MAX` gas can be used to send as much as
    /// possible.
    ///
    /// [`DELEGATE_CALL`]: https://www.evm.codes/#F4
    fn delegate_call_contract(
        &mut self,
        contract: GuestPtr,
        data: GuestPtr,
        data_len: u32,
        gas: Gas,
        ret_len: GuestPtr,
    ) -> Result<u8, Self::Err> {
        self.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
        self.pay_for_read(data_len)?;
        self.pay_for_read(data_len)?; // read from geth

        let CallInputs {
            contract,
            input,
            gas_left,
            gas_req,
            ..
        } = self.parse_call_inputs(contract, data, gas, data_len, None)?;

        let (outs_len, gas_cost, status) = self
            .evm_api()
            .delegate_call(contract, &input, gas_left, gas_req)?;

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = outs_len;
        self.write_u32(ret_len, outs_len)?;
        let status = status as u8;

        if self.evm_data().tracing {
            return trace!(
                "delegate_call_contract",
                self,
                [contract, be!(gas), &input],
                [be!(outs_len), be!(status)],
                status
            );
        }
        Ok(status)
    }

    /// Static calls the contract at the given address, with the option to limit the amount of gas
    /// supplied. The return status indicates whether the call succeeded, and is nonzero on
    /// failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which can
    /// be read via the `read_return_data` hostio. The bytes are not returned directly so that the
    /// programmer can potentially save gas by choosing which subset of the return result they'd
    /// like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`STATIC_CALL`] opcode, including the
    /// 63/64 gas rule. This means that supplying `u64::MAX` gas can be used to send as much as
    /// possible.
    ///
    /// [`STATIC_CALL`]: https://www.evm.codes/#FA
    fn static_call_contract(
        &mut self,
        contract: GuestPtr,
        data: GuestPtr,
        data_len: u32,
        gas: Gas,
        ret_len: GuestPtr,
    ) -> Result<u8, Self::Err> {
        self.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
        self.pay_for_read(data_len)?;
        self.pay_for_read(data_len)?; // read from geth

        let CallInputs {
            contract,
            input,
            gas_left,
            gas_req,
            ..
        } = self.parse_call_inputs(contract, data, gas, data_len, None)?;

        let (outs_len, gas_cost, status) = self
            .evm_api()
            .static_call(contract, &input, gas_left, gas_req)?;

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = outs_len;
        self.write_u32(ret_len, outs_len)?;
        let status = status as u8;

        if self.evm_data().tracing {
            return trace!(
                "static_call_contract",
                self,
                [contract, be!(gas), &input],
                [be!(outs_len), be!(status)],
                status
            );
        }
        Ok(status)
    }

    /// Deploys a new contract using the init code provided, which the EVM executes to construct
    /// the code of the newly deployed contract. The init code must be written in EVM bytecode, but
    /// the code it deploys can be that of a Stylus program. The code returned will be treated as
    /// WASM if it begins with the EOF-inspired header `0xEFF000`. Otherwise the code will be
    /// interpreted as that of a traditional EVM-style contract. See [`Deploying Stylus Programs`]
    /// for more information on writing init code.
    ///
    /// On success, this hostio returns the address of the newly created account whose address is
    /// a function of the sender and nonce. On failure the address will be `0`, `return_data_len`
    /// will store the length of the revert data, the bytes of which can be read via the
    /// `read_return_data` hostio. The semantics are equivalent to that of the EVM's [`CREATE`]
    /// opcode, which notably includes the exact address returned.
    ///
    /// [`Deploying Stylus Programs`]: https://docs.arbitrum.io/stylus/quickstart#deploying-your-contract
    /// [`CREATE`]: https://www.evm.codes/#f0
    fn create1(
        &mut self,
        code: GuestPtr,
        code_len: u32,
        endowment: GuestPtr,
        contract: GuestPtr,
        revert_data_len: GuestPtr,
    ) -> Result<(), Self::Err> {
        self.buy_ink(hostio::CREATE1_BASE_INK)?;
        self.pay_for_read(code_len)?;
        self.pay_for_read(code_len)?; // read from geth

        let code = self.read_slice(code, code_len)?;
        let code_copy = self.evm_data().tracing.then(|| code.clone());

        let endowment = self.read_bytes32(endowment)?;
        let gas = self.gas_left()?;
        let api = self.evm_api();

        let (response, ret_len, gas_cost) = api.create1(code, endowment, gas)?;

        let address = match response {
            CreateRespone::Fail(reason) => return Err(eyre!(reason).into()),
            CreateRespone::Succes(addr) => addr,
        };

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = ret_len;
        self.write_u32(revert_data_len, ret_len)?;
        self.write_bytes20(contract, address)?;

        trace!(
            "create1",
            self,
            [endowment, code_copy.unwrap()],
            [address, be!(ret_len)],
            ()
        )
    }

    /// Deploys a new contract using the init code provided, which the EVM executes to construct
    /// the code of the newly deployed contract. The init code must be written in EVM bytecode, but
    /// the code it deploys can be that of a Stylus program. The code returned will be treated as
    /// WASM if it begins with the EOF-inspired header `0xEFF000`. Otherwise the code will be
    /// interpreted as that of a traditional EVM-style contract. See [`Deploying Stylus Programs`]
    /// for more information on writing init code.
    ///
    /// On success, this hostio returns the address of the newly created account whose address is a
    /// function of the sender, salt, and init code. On failure the address will be `0`,
    /// `return_data_len` will store the length of the revert data, the bytes of which can be read
    /// via the `read_return_data` hostio. The semantics are equivalent to that of the EVM's
    /// `[CREATE2`] opcode, which notably includes the exact address returned.
    ///
    /// [`Deploying Stylus Programs`]: https://docs.arbitrum.io/stylus/quickstart#deploying-your-contract
    /// [`CREATE2`]: https://www.evm.codes/#f5
    fn create2(
        &mut self,
        code: GuestPtr,
        code_len: u32,
        endowment: GuestPtr,
        salt: GuestPtr,
        contract: GuestPtr,
        revert_data_len: GuestPtr,
    ) -> Result<(), Self::Err> {
        self.buy_ink(hostio::CREATE2_BASE_INK)?;
        self.pay_for_read(code_len)?;
        self.pay_for_read(code_len)?; // read from geth

        let code = self.read_slice(code, code_len)?;
        let code_copy = self.evm_data().tracing.then(|| code.clone());

        let endowment = self.read_bytes32(endowment)?;
        let salt = self.read_bytes32(salt)?;
        let gas = self.gas_left()?;
        let api = self.evm_api();

        let (response, ret_len, gas_cost) = api.create2(code, endowment, salt, gas)?;

        let address = match response {
            CreateRespone::Fail(reason) => return Err(eyre!(reason).into()),
            CreateRespone::Succes(addr) => addr,
        };

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = ret_len;
        self.write_u32(revert_data_len, ret_len)?;
        self.write_bytes20(contract, address)?;

        let salt = salt.into_iter();
        trace!(
            "create2",
            self,
            [endowment, salt, code_copy.unwrap()],
            [address, be!(ret_len)],
            ()
        )
    }

    /// Copies the bytes of the last EVM call or deployment return result. Does not revert if out of
    /// bounds, but rather copies the overlapping portion. The semantics are otherwise equivalent
    /// to that of the EVM's [`RETURN_DATA_COPY`] opcode.
    ///
    /// Returns the number of bytes written.
    ///
    /// [`RETURN_DATA_COPY`]: https://www.evm.codes/#3e
    fn read_return_data(
        &mut self,
        dest: GuestPtr,
        offset: u32,
        size: u32,
    ) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::READ_RETURN_DATA_BASE_INK)?;

        // pay for only as many bytes as could possibly be written
        let max = self.evm_return_data_len().saturating_sub(offset);
        self.pay_for_write(size.min(max))?;
        if max == 0 {
            // no return data, nothing to write
            return trace!("read_return_data", self, [be!(offset), be!(size)], &[], 0);
        }

        let ret_data = self.evm_api().get_return_data();
        let ret_data = ret_data.slice();
        let out_slice = arbutil::slice_with_runoff(&ret_data, offset, offset.saturating_add(size));

        let out_len = out_slice.len() as u32;
        if out_len > 0 {
            self.write_slice(dest, out_slice)?;
        }
        trace!(
            "read_return_data",
            self,
            [be!(offset), be!(size)],
            out_slice.to_vec(),
            out_len
        )
    }

    /// Returns the length of the last EVM call or deployment return result, or `0` if neither have
    /// happened during the program's execution. The semantics are equivalent to that of the EVM's
    /// [`RETURN_DATA_SIZE`] opcode.
    ///
    /// [`RETURN_DATA_SIZE`]: https://www.evm.codes/#3d
    fn return_data_size(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::RETURN_DATA_SIZE_BASE_INK)?;
        let len = *self.evm_return_data_len();
        trace!("return_data_size", self, &[], be!(len), len)
    }

    /// Emits an EVM log with the given number of topics and data, the first bytes of which should
    /// be the 32-byte-aligned topic data. The semantics are equivalent to that of the EVM's
    /// [`LOG0`], [`LOG1`], [`LOG2`], [`LOG3`], and [`LOG4`] opcodes based on the number of topics
    /// specified. Requesting more than `4` topics will induce a revert.
    ///
    /// [`LOG0`]: https://www.evm.codes/#a0
    /// [`LOG1`]: https://www.evm.codes/#a1
    /// [`LOG2`]: https://www.evm.codes/#a2
    /// [`LOG3`]: https://www.evm.codes/#a3
    /// [`LOG4`]: https://www.evm.codes/#a4
    fn emit_log(&mut self, data: GuestPtr, len: u32, topics: u32) -> Result<(), Self::Err> {
        self.buy_ink(hostio::EMIT_LOG_BASE_INK)?;
        if topics > 4 || len < topics * 32 {
            Err(eyre!("bad topic data"))?;
        }
        self.pay_for_read(len)?;
        self.pay_for_evm_log(topics, len - topics * 32)?;

        let data = self.read_slice(data, len)?;
        self.evm_api().emit_log(data.clone(), topics)?;
        trace!("emit_log", self, [be!(topics), data], &[])
    }

    /// Gets the ETH balance in wei of the account at the given address.
    /// The semantics are equivalent to that of the EVM's [`BALANCE`] opcode.
    ///
    /// [`BALANCE`]: https://www.evm.codes/#31
    fn account_balance(&mut self, address: GuestPtr, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::ACCOUNT_BALANCE_BASE_INK)?;
        self.require_gas(evm::COLD_ACCOUNT_GAS)?;
        let address = self.read_bytes20(address)?;

        let (balance, gas_cost) = self.evm_api().account_balance(address)?;
        self.buy_gas(gas_cost)?;
        self.write_bytes32(ptr, balance)?;
        trace!("account_balance", self, address, balance)
    }

    /// Gets a subset of the code from the account at the given address. The semantics are identical to that
    /// of the EVM's [`EXT_CODE_COPY`] opcode, aside from one small detail: the write to the buffer `dest` will
    /// stop after the last byte is written. This is unlike the EVM, which right pads with zeros in this scenario.
    /// The return value is the number of bytes written, which allows the caller to detect if this has occured.
    ///
    /// [`EXT_CODE_COPY`]: https://www.evm.codes/#3C
    fn account_code(
        &mut self,
        address: GuestPtr,
        offset: u32,
        size: u32,
        dest: GuestPtr,
    ) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::ACCOUNT_CODE_BASE_INK)?;
        self.require_gas(evm::COLD_ACCOUNT_GAS)?; // not necessary since we also check in Go

        let address = self.read_bytes20(address)?;
        let gas = self.gas_left()?;

        let arbos_version = self.evm_data().arbos_version;

        // we pass `gas` to check if there's enough before loading from the db
        let (code, gas_cost) = self.evm_api().account_code(arbos_version, address, gas)?;
        self.buy_gas(gas_cost)?;

        let code = code.slice();
        self.pay_for_write(code.len() as u32)?;

        let out_slice = arbutil::slice_with_runoff(&code, offset, offset.saturating_add(size));
        let out_len = out_slice.len() as u32;
        self.write_slice(dest, out_slice)?;

        trace!(
            "account_code",
            self,
            [address, be!(offset), be!(size)],
            out_slice.to_vec(),
            out_len
        )
    }

    /// Gets the size of the code in bytes at the given address. The semantics are equivalent
    /// to that of the EVM's [`EXT_CODESIZE`].
    ///
    /// [`EXT_CODESIZE`]: https://www.evm.codes/#3B
    fn account_code_size(&mut self, address: GuestPtr) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::ACCOUNT_CODE_SIZE_BASE_INK)?;
        self.require_gas(evm::COLD_ACCOUNT_GAS)?; // not necessary since we also check in Go
        let address = self.read_bytes20(address)?;
        let gas = self.gas_left()?;

        let arbos_version = self.evm_data().arbos_version;

        // we pass `gas` to check if there's enough before loading from the db
        let (code, gas_cost) = self.evm_api().account_code(arbos_version, address, gas)?;
        self.buy_gas(gas_cost)?;

        let code = code.slice();
        let len = code.len() as u32;
        trace!("account_code_size", self, address, be!(len), len)
    }

    /// Gets the code hash of the account at the given address. The semantics are equivalent
    /// to that of the EVM's [`EXT_CODEHASH`] opcode. Note that the code hash of an account without
    /// code will be the empty hash
    /// `keccak("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470`.
    ///
    /// [`EXT_CODEHASH`]: https://www.evm.codes/#3F
    fn account_codehash(&mut self, address: GuestPtr, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::ACCOUNT_CODE_HASH_BASE_INK)?;
        self.require_gas(evm::COLD_ACCOUNT_GAS)?;
        let address = self.read_bytes20(address)?;

        let (hash, gas_cost) = self.evm_api().account_codehash(address)?;
        self.buy_gas(gas_cost)?;
        self.write_bytes32(ptr, hash)?;
        trace!("account_codehash", self, address, hash)
    }

    /// Gets the basefee of the current block. The semantics are equivalent to that of the EVM's
    /// [`BASEFEE`] opcode.
    ///
    /// [`BASEFEE`]: https://www.evm.codes/#48
    fn block_basefee(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::BLOCK_BASEFEE_BASE_INK)?;
        self.write_bytes32(ptr, self.evm_data().block_basefee)?;
        trace!("block_basefee", self, &[], self.evm_data().block_basefee)
    }

    /// Gets the coinbase of the current block, which on Arbitrum chains is the L1 batch poster's
    /// address. This differs from Ethereum where the validator including the transaction
    /// determines the coinbase.
    fn block_coinbase(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::BLOCK_COINBASE_BASE_INK)?;
        self.write_bytes20(ptr, self.evm_data().block_coinbase)?;
        trace!("block_coinbase", self, &[], self.evm_data().block_coinbase)
    }

    /// Gets the gas limit of the current block. The semantics are equivalent to that of the EVM's
    /// [`GAS_LIMIT`] opcode. Note that as of the time of this writing, `evm.codes` incorrectly
    /// implies that the opcode returns the gas limit of the current transaction.  When in doubt,
    /// consult [`The Ethereum Yellow Paper`].
    ///
    /// [`GAS_LIMIT`]: https://www.evm.codes/#45
    /// [`The Ethereum Yellow Paper`]: https://ethereum.github.io/yellowpaper/paper.pdf
    fn block_gas_limit(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(hostio::BLOCK_GAS_LIMIT_BASE_INK)?;
        let limit = self.evm_data().block_gas_limit;
        trace!("block_gas_limit", self, &[], be!(limit), limit)
    }

    /// Gets a bounded estimate of the L1 block number at which the Sequencer sequenced the
    /// transaction. See [`Block Numbers and Time`] for more information on how this value is
    /// determined.
    ///
    /// [`Block Numbers and Time`]: https://developer.arbitrum.io/time
    fn block_number(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(hostio::BLOCK_NUMBER_BASE_INK)?;
        let number = self.evm_data().block_number;
        trace!("block_number", self, &[], be!(number), number)
    }

    /// Gets a bounded estimate of the Unix timestamp at which the Sequencer sequenced the
    /// transaction. See [`Block Numbers and Time`] for more information on how this value is
    /// determined.
    ///
    /// [`Block Numbers and Time`]: https://developer.arbitrum.io/time
    fn block_timestamp(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(hostio::BLOCK_TIMESTAMP_BASE_INK)?;
        let timestamp = self.evm_data().block_timestamp;
        trace!("block_timestamp", self, &[], be!(timestamp), timestamp)
    }

    /// Gets the unique chain identifier of the Arbitrum chain. The semantics are equivalent to
    /// that of the EVM's [`CHAIN_ID`] opcode.
    ///
    /// [`CHAIN_ID`]: https://www.evm.codes/#46
    fn chainid(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(hostio::CHAIN_ID_BASE_INK)?;
        let chainid = self.evm_data().chainid;
        trace!("chainid", self, &[], be!(chainid), chainid)
    }

    /// Gets the address of the current program. The semantics are equivalent to that of the EVM's
    /// [`ADDRESS`] opcode.
    ///
    /// [`ADDRESS`]: https://www.evm.codes/#30
    fn contract_address(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::ADDRESS_BASE_INK)?;
        self.write_bytes20(ptr, self.evm_data().contract_address)?;
        trace!(
            "contract_address",
            self,
            &[],
            self.evm_data().contract_address
        )
    }

    /// Gets the amount of gas left after paying for the cost of this hostio. The semantics are
    /// equivalent to that of the EVM's [`GAS`] opcode.
    ///
    /// [`GAS`]: https://www.evm.codes/#5a
    fn evm_gas_left(&mut self) -> Result<Gas, Self::Err> {
        self.buy_ink(hostio::EVM_GAS_LEFT_BASE_INK)?;
        let gas = self.gas_left()?;
        trace!("evm_gas_left", self, &[], be!(gas), gas)
    }

    /// Gets the amount of ink remaining after paying for the cost of this hostio. The semantics
    /// are equivalent to that of the EVM's [`GAS`] opcode, except the units are in ink. See
    /// [`Ink and Gas`] for more information on Stylus's compute pricing.
    ///
    /// [`GAS`]: https://www.evm.codes/#5a
    /// [`Ink and Gas`]: https://docs.arbitrum.io/stylus/concepts/gas-metering#ink-and-gas
    fn evm_ink_left(&mut self) -> Result<Ink, Self::Err> {
        self.buy_ink(hostio::EVM_INK_LEFT_BASE_INK)?;
        let ink = self.ink_ready()?;
        trace!("evm_ink_left", self, &[], be!(ink), ink)
    }

    /// Computes `value ÷ exponent` using 256-bit math, writing the result to the first.
    /// The semantics are equivalent to that of the EVM's [`DIV`] opcode, which means that a `divisor` of `0`
    /// writes `0` to `value`.
    ///
    /// [`DIV`]: https://www.evm.codes/#04
    fn math_div(&mut self, value: GuestPtr, divisor: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MATH_DIV_BASE_INK)?;
        let (a, a32) = self.read_u256(value)?;
        let (b, b32) = self.read_u256(divisor)?;

        let result = a.checked_div(b).unwrap_or_default().into();
        self.write_bytes32(value, result)?;
        trace!("math_div", self, [a32, b32], result)
    }

    /// Computes `value % exponent` using 256-bit math, writing the result to the first.
    /// The semantics are equivalent to that of the EVM's [`MOD`] opcode, which means that a `modulus` of `0`
    /// writes `0` to `value`.
    ///
    /// [`MOD`]: https://www.evm.codes/#06
    fn math_mod(&mut self, value: GuestPtr, modulus: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MATH_MOD_BASE_INK)?;
        let (a, a32) = self.read_u256(value)?;
        let (b, b32) = self.read_u256(modulus)?;

        let result = a.checked_rem(b).unwrap_or_default().into();
        self.write_bytes32(value, result)?;
        trace!("math_mod", self, [a32, b32], result)
    }

    /// Computes `value ^ exponent` using 256-bit math, writing the result to the first.
    /// The semantics are equivalent to that of the EVM's [`EXP`] opcode.
    ///
    /// [`EXP`]: https://www.evm.codes/#0A
    fn math_pow(&mut self, value: GuestPtr, exponent: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MATH_POW_BASE_INK)?;
        let (a, a32) = self.read_u256(value)?;
        let (b, b32) = self.read_u256(exponent)?;

        self.pay_for_pow(&b32)?;
        let result = a.wrapping_pow(b).into();
        self.write_bytes32(value, result)?;
        trace!("math_pow", self, [a32, b32], result)
    }

    /// Computes `(value + addend) % modulus` using 256-bit math, writing the result to the first.
    /// The semantics are equivalent to that of the EVM's [`ADDMOD`] opcode, which means that a `modulus` of `0`
    /// writes `0` to `value`.
    ///
    /// [`ADDMOD`]: https://www.evm.codes/#08
    fn math_add_mod(
        &mut self,
        value: GuestPtr,
        addend: GuestPtr,
        modulus: GuestPtr,
    ) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MATH_ADD_MOD_BASE_INK)?;
        let (a, a32) = self.read_u256(value)?;
        let (b, b32) = self.read_u256(addend)?;
        let (c, c32) = self.read_u256(modulus)?;

        let result = a.add_mod(b, c).into();
        self.write_bytes32(value, result)?;
        trace!("math_add_mod", self, [a32, b32, c32], result)
    }

    /// Computes `(value * multiplier) % modulus` using 256-bit math, writing the result to the first.
    /// The semantics are equivalent to that of the EVM's [`MULMOD`] opcode, which means that a `modulus` of `0`
    /// writes `0` to `value`.
    ///
    /// [`MULMOD`]: https://www.evm.codes/#09
    fn math_mul_mod(
        &mut self,
        value: GuestPtr,
        multiplier: GuestPtr,
        modulus: GuestPtr,
    ) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MATH_MUL_MOD_BASE_INK)?;
        let (a, a32) = self.read_u256(value)?;
        let (b, b32) = self.read_u256(multiplier)?;
        let (c, c32) = self.read_u256(modulus)?;

        let result = a.mul_mod(b, c).into();
        self.write_bytes32(value, result)?;
        trace!("math_mul_mod", self, [a32, b32, c32], result)
    }

    /// Whether the current call is reentrant.
    fn msg_reentrant(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::MSG_REENTRANT_BASE_INK)?;
        let reentrant = self.evm_data().reentrant;
        trace!("msg_reentrant", self, &[], be!(reentrant), reentrant)
    }

    /// Gets the address of the account that called the program. For normal L2-to-L2 transactions
    /// the semantics are equivalent to that of the EVM's [`CALLER`] opcode, including in cases
    /// arising from [`DELEGATE_CALL`].
    ///
    /// For L1-to-L2 retryable ticket transactions, the top-level sender's address will be aliased.
    /// See [`Retryable Ticket Address Aliasing`][aliasing] for more information on how this works.
    ///
    /// [`CALLER`]: https://www.evm.codes/#33
    /// [`DELEGATE_CALL`]: https://www.evm.codes/#f4
    /// [aliasing]: https://developer.arbitrum.io/arbos/l1-to-l2-messaging#address-aliasing
    fn msg_sender(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MSG_SENDER_BASE_INK)?;
        self.write_bytes20(ptr, self.evm_data().msg_sender)?;
        trace!("msg_sender", self, &[], self.evm_data().msg_sender)
    }

    /// Get the ETH value in wei sent to the program. The semantics are equivalent to that of the
    /// EVM's [`CALLVALUE`] opcode.
    ///
    /// [`CALLVALUE`]: https://www.evm.codes/#34
    fn msg_value(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::MSG_VALUE_BASE_INK)?;
        self.write_bytes32(ptr, self.evm_data().msg_value)?;
        trace!("msg_value", self, &[], self.evm_data().msg_value)
    }

    /// Efficiently computes the [`keccak256`] hash of the given preimage.
    /// The semantics are equivalent to that of the EVM's [`SHA3`] opcode.
    ///
    /// [`keccak256`]: https://en.wikipedia.org/wiki/SHA-3
    /// [`SHA3`]: https://www.evm.codes/#20
    fn native_keccak256(
        &mut self,
        input: GuestPtr,
        len: u32,
        output: GuestPtr,
    ) -> Result<(), Self::Err> {
        self.pay_for_keccak(len)?;
        let preimage = self.read_slice(input, len)?;
        let digest = crypto::keccak(&preimage);
        self.write_bytes32(output, digest.into())?;
        trace!("native_keccak256", self, preimage, digest)
    }

    /// Gets the gas price in wei per gas, which on Arbitrum chains equals the basefee. The
    /// semantics are equivalent to that of the EVM's [`GAS_PRICE`] opcode.
    ///
    /// [`GAS_PRICE`]: https://www.evm.codes/#3A
    fn tx_gas_price(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::TX_GAS_PRICE_BASE_INK)?;
        self.write_bytes32(ptr, self.evm_data().tx_gas_price)?;
        trace!("tx_gas_price", self, &[], self.evm_data().tx_gas_price)
    }

    /// Gets the price of ink in evm gas basis points. See [`Ink and Gas`] for more information on
    /// Stylus's compute-pricing model.
    ///
    /// [`Ink and Gas`]: https://docs.arbitrum.io/stylus/concepts/gas-metering#ink-and-gas
    fn tx_ink_price(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(hostio::TX_INK_PRICE_BASE_INK)?;
        let ink_price = self.pricing().ink_price;
        trace!("tx_ink_price", self, &[], be!(ink_price), ink_price)
    }

    /// Gets the top-level sender of the transaction. The semantics are equivalent to that of the
    /// EVM's [`ORIGIN`] opcode.
    ///
    /// [`ORIGIN`]: https://www.evm.codes/#32
    fn tx_origin(&mut self, ptr: GuestPtr) -> Result<(), Self::Err> {
        self.buy_ink(hostio::TX_ORIGIN_BASE_INK)?;
        self.write_bytes20(ptr, self.evm_data().tx_origin)?;
        trace!("tx_origin", self, &[], self.evm_data().tx_origin)
    }

    /// Pays for new pages as needed before the memory.grow opcode is invoked.
    fn pay_for_memory_grow(&mut self, pages: u16) -> Result<(), Self::Err> {
        if pages == 0 {
            self.buy_ink(hostio::PAY_FOR_MEMORY_GROW_BASE_INK)?;
            return trace!("pay_for_memory_grow", self, be!(pages), &[]);
        }
        let gas_cost = self.evm_api().add_pages(pages)?; // no sentry needed since the work happens after the hostio
        self.buy_gas(gas_cost)?;
        trace!("pay_for_memory_grow", self, be!(pages), &[])
    }

    /// Prints a UTF-8 encoded string to the console. Only available in debug mode.
    fn console_log_text(&mut self, ptr: GuestPtr, len: u32) -> Result<(), Self::Err> {
        let text = self.read_slice(ptr, len)?;
        self.say(String::from_utf8_lossy(&text));
        trace!("console_log_text", self, text, &[])
    }

    /// Prints a value to the console. Only available in debug mode.
    fn console_log<T: Into<Value>>(&mut self, value: T) -> Result<(), Self::Err> {
        let value = value.into();
        self.say(value);
        trace!("console_log", self, [format!("{value}").as_bytes()], &[])
    }

    /// Prints and returns a value to the console. Only available in debug mode.
    fn console_tee<T: Into<Value> + Copy>(&mut self, value: T) -> Result<T, Self::Err> {
        self.say(value.into());
        Ok(value)
    }

    // Initializes benchmark data related to a code block.
    // A code block is defined by the instructions between start_benchmark and end_benchmark calls.
    // If start_benchmark is called multiple times without end_benchmark being called,
    // then only the last start_benchmark before end_benchmark will be used.
    // It is possible to have multiple code blocks benchmarked in the same program.
    fn start_benchmark(&mut self) -> Result<(), Self::Err> {
        let ink_curr = self.ink_ready()?;

        let benchmark = self.benchmark();
        benchmark.timer = Some(Instant::now());
        benchmark.ink_start = Some(ink_curr);

        Ok(())
    }

    // Updates cumulative benchmark data related to a code block.
    // If end_benchmark is called without a corresponding start_benchmark nothing will happen.
    fn end_benchmark(&mut self) -> Result<(), Self::Err> {
        let ink_curr = self.ink_ready()?;

        let benchmark = self.benchmark();
        if let Some(timer) = benchmark.timer {
            benchmark.elapsed_total = benchmark.elapsed_total.saturating_add(timer.elapsed());

            let code_block_ink = benchmark.ink_start.unwrap().saturating_sub(ink_curr);
            benchmark.ink_total = benchmark.ink_total.saturating_add(code_block_ink);

            benchmark.timer = None;
            benchmark.ink_start = None;
        };

        Ok(())
    }
}
