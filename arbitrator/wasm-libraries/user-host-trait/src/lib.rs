// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{
    crypto,
    evm::{self, api::EvmApi, user::UserOutcomeKind, EvmData},
    pricing::{EVM_API_INK, HOSTIO_INK, PTR_INK},
    Bytes20, Bytes32,
};
use eyre::{eyre, Result};
use prover::{
    programs::{meter::OutOfInkError, prelude::*},
    value::Value,
};
use std::fmt::Display;

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

#[allow(clippy::too_many_arguments)]
pub trait UserHost: GasMeteredMachine {
    type Err: From<OutOfInkError> + From<Self::MemoryErr> + From<eyre::ErrReport>;
    type MemoryErr;
    type A: EvmApi;

    fn args(&self) -> &[u8];
    fn outs(&mut self) -> &mut Vec<u8>;

    fn evm_api(&mut self) -> &mut Self::A;
    fn evm_data(&self) -> &EvmData;
    fn evm_return_data_len(&mut self) -> &mut u32;

    fn read_bytes20(&self, ptr: u32) -> Result<Bytes20, Self::MemoryErr>;
    fn read_bytes32(&self, ptr: u32) -> Result<Bytes32, Self::MemoryErr>;
    fn read_slice(&self, ptr: u32, len: u32) -> Result<Vec<u8>, Self::MemoryErr>;

    fn write_u32(&mut self, ptr: u32, x: u32) -> Result<(), Self::MemoryErr>;
    fn write_bytes20(&self, ptr: u32, src: Bytes20) -> Result<(), Self::MemoryErr>;
    fn write_bytes32(&self, ptr: u32, src: Bytes32) -> Result<(), Self::MemoryErr>;
    fn write_slice(&self, ptr: u32, src: &[u8]) -> Result<(), Self::MemoryErr>;

    fn say<D: Display>(&self, text: D);
    fn trace(&self, name: &str, args: &[u8], outs: &[u8], end_ink: u64);

    fn read_args(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        self.pay_for_write(self.args().len() as u32)?;
        self.write_slice(ptr, self.args())?;
        trace!("read_args", self, &[], self.args())
    }

    fn write_result(&mut self, ptr: u32, len: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        self.pay_for_read(len)?;
        *self.outs() = self.read_slice(ptr, len)?;
        trace!("write_result", self, &*self.outs(), &[])
    }

    fn storage_load_bytes32(&mut self, key: u32, dest: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + 2 * PTR_INK + EVM_API_INK)?;
        let key = self.read_bytes32(key)?;

        let (value, gas_cost) = self.evm_api().get_bytes32(key);
        self.buy_gas(gas_cost)?;
        self.write_bytes32(dest, value)?;
        trace!("storage_load_bytes32", self, key, value)
    }

    fn storage_store_bytes32(&mut self, key: u32, value: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + 2 * PTR_INK + EVM_API_INK)?;
        self.require_gas(evm::SSTORE_SENTRY_GAS)?; // see operations_acl_arbitrum.go

        let key = self.read_bytes32(key)?;
        let value = self.read_bytes32(value)?;

        let gas_cost = self.evm_api().set_bytes32(key, value)?;
        self.buy_gas(gas_cost)?;
        trace!("storage_store_bytes32", self, [key, value], &[])
    }

    fn call_contract(
        &mut self,
        contract: u32,
        data: u32,
        data_len: u32,
        value: u32,
        gas: u64,
        ret_len: u32,
    ) -> Result<u8, Self::Err> {
        let value = Some(value);
        let call = |api: &mut Self::A, contract, data: &_, gas, value: Option<_>| {
            api.contract_call(contract, data, gas, value.unwrap())
        };
        self.do_call(contract, data, data_len, value, gas, ret_len, call, "")
    }

    fn delegate_call_contract(
        &mut self,
        contract: u32,
        data: u32,
        data_len: u32,
        gas: u64,
        ret_len: u32,
    ) -> Result<u8, Self::Err> {
        let call =
            |api: &mut Self::A, contract, data: &_, gas, _| api.delegate_call(contract, data, gas);
        self.do_call(
            contract, data, data_len, None, gas, ret_len, call, "delegate",
        )
    }

    fn static_call_contract(
        &mut self,
        contract: u32,
        data: u32,
        data_len: u32,
        gas: u64,
        ret_len: u32,
    ) -> Result<u8, Self::Err> {
        let call =
            |api: &mut Self::A, contract, data: &_, gas, _| api.static_call(contract, data, gas);
        self.do_call(contract, data, data_len, None, gas, ret_len, call, "static")
    }

    fn do_call<F>(
        &mut self,
        contract: u32,
        calldata: u32,
        calldata_len: u32,
        value: Option<u32>,
        mut gas: u64,
        return_data_len: u32,
        call: F,
        name: &str,
    ) -> Result<u8, Self::Err>
    where
        F: FnOnce(&mut Self::A, Address, &[u8], u64, Option<Wei>) -> (u32, u64, UserOutcomeKind),
    {
        self.buy_ink(HOSTIO_INK + 3 * PTR_INK + EVM_API_INK)?;
        self.pay_for_read(calldata_len)?;

        let gas_passed = gas;
        gas = gas.min(self.gas_left()?); // provide no more than what the user has

        let contract = self.read_bytes20(contract)?;
        let input = self.read_slice(calldata, calldata_len)?;
        let value = value.map(|x| self.read_bytes32(x)).transpose()?;
        let api = self.evm_api();

        let (outs_len, gas_cost, status) = call(api, contract, &input, gas, value);
        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = outs_len;
        self.write_u32(return_data_len, outs_len)?;
        let status = status as u8;

        if self.evm_data().tracing {
            let underscore = (!name.is_empty()).then_some("_").unwrap_or_default();
            let name = format!("{name}{underscore}call_contract");
            let value = value.into_iter().flatten();
            return trace!(
                &name,
                self,
                [contract, be!(gas_passed), value, &input],
                [be!(outs_len), be!(status)],
                status
            );
        }
        Ok(status)
    }

    fn create1(
        &mut self,
        code: u32,
        code_len: u32,
        endowment: u32,
        contract: u32,
        revert_data_len: u32,
    ) -> Result<(), Self::Err> {
        let call = |api: &mut Self::A, code, value, _, gas| api.create1(code, value, gas);
        self.do_create(
            code,
            code_len,
            endowment,
            None,
            contract,
            revert_data_len,
            3 * PTR_INK + EVM_API_INK,
            call,
            "create1",
        )
    }

    fn create2(
        &mut self,
        code: u32,
        code_len: u32,
        endowment: u32,
        salt: u32,
        contract: u32,
        revert_data_len: u32,
    ) -> Result<(), Self::Err> {
        let call = |api: &mut Self::A, code, value, salt: Option<_>, gas| {
            api.create2(code, value, salt.unwrap(), gas)
        };
        self.do_create(
            code,
            code_len,
            endowment,
            Some(salt),
            contract,
            revert_data_len,
            4 * PTR_INK + EVM_API_INK,
            call,
            "create2",
        )
    }

    fn do_create<F>(
        &mut self,
        code: u32,
        code_len: u32,
        endowment: u32,
        salt: Option<u32>,
        contract: u32,
        revert_data_len: u32,
        cost: u64,
        call: F,
        name: &str,
    ) -> Result<(), Self::Err>
    where
        F: FnOnce(&mut Self::A, Vec<u8>, Bytes32, Option<Wei>, u64) -> (Result<Address>, u32, u64),
    {
        self.buy_ink(HOSTIO_INK + cost)?;
        self.pay_for_read(code_len)?;

        let code = self.read_slice(code, code_len)?;
        let code_copy = self.evm_data().tracing.then(|| code.clone());

        let endowment = self.read_bytes32(endowment)?;
        let salt = salt.map(|x| self.read_bytes32(x)).transpose()?;
        let gas = self.gas_left()?;
        let api = self.evm_api();

        let (result, ret_len, gas_cost) = call(api, code, endowment, salt, gas);
        let result = result?;

        self.buy_gas(gas_cost)?;
        *self.evm_return_data_len() = ret_len;
        self.write_u32(revert_data_len, ret_len)?;
        self.write_bytes20(contract, result)?;

        let salt = salt.into_iter().flatten();
        trace!(
            name,
            self,
            [endowment, salt, code_copy.unwrap()],
            [result, be!(ret_len)],
            ()
        )
    }

    fn read_return_data(&mut self, dest: u32, offset: u32, size: u32) -> Result<u32, Self::Err> {
        self.buy_ink(HOSTIO_INK + EVM_API_INK)?;
        self.pay_for_write(size)?;

        let data = self.evm_api().get_return_data(offset, size);
        assert!(data.len() <= size as usize);
        self.write_slice(dest, &data)?;

        let len = data.len() as u32;
        trace!(
            "read_return_data",
            self,
            [be!(offset), be!(size)],
            data,
            len
        )
    }

    fn return_data_size(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let len = *self.evm_return_data_len();
        trace!("return_data_size", self, be!(len), &[], len)
    }

    fn emit_log(&mut self, data: u32, len: u32, topics: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + EVM_API_INK)?;
        if topics > 4 || len < topics * 32 {
            println!("too many!!!!!!!!!!!!!!!!");
            Err(eyre!("bad topic data"))?;
        }
        self.pay_for_read(len)?;
        self.pay_for_evm_log(topics, len - topics * 32)?;

        let data = self.read_slice(data, len)?;
        self.evm_api().emit_log(data.clone(), topics)?;
        trace!("emit_log", self, [be!(topics), data], &[])
    }

    fn account_balance(&mut self, address: u32, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + 2 * PTR_INK + EVM_API_INK)?;
        let address = self.read_bytes20(address)?;

        let (balance, gas_cost) = self.evm_api().account_balance(address);
        self.buy_gas(gas_cost)?;
        self.write_bytes32(ptr, balance)?;
        trace!("account_balance", self, address, balance)
    }

    fn account_codehash(&mut self, address: u32, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + 2 * PTR_INK + EVM_API_INK)?;
        let address = self.read_bytes20(address)?;

        let (hash, gas_cost) = self.evm_api().account_codehash(address);
        self.buy_gas(gas_cost)?;
        self.write_bytes32(ptr, hash)?;
        trace!("account_codehash", self, address, hash)
    }

    fn block_basefee(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes32(ptr, self.evm_data().block_basefee)?;
        trace!("block_basefee", self, &[], self.evm_data().block_basefee)
    }

    fn block_coinbase(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes20(ptr, self.evm_data().block_coinbase)?;
        trace!("block_coinbase", self, &[], self.evm_data().block_coinbase)
    }

    fn block_gas_limit(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let limit = self.evm_data().block_gas_limit;
        trace!("block_gas_limit", self, &[], be!(limit), limit)
    }

    fn block_number(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let number = self.evm_data().block_number;
        trace!("block_number", self, &[], be!(number), number)
    }

    fn block_timestamp(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let timestamp = self.evm_data().block_timestamp;
        trace!("block_timestamp", self, &[], be!(timestamp), timestamp)
    }

    fn chainid(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let chainid = self.evm_data().chainid;
        trace!("chainid", self, &[], be!(chainid), chainid)
    }

    fn contract_address(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes20(ptr, self.evm_data().contract_address)?;
        trace!(
            "contract_address",
            self,
            &[],
            self.evm_data().contract_address
        )
    }

    fn evm_gas_left(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let gas = self.gas_left()?;
        trace!("evm_gas_left", self, be!(gas), &[], gas)
    }

    fn evm_ink_left(&mut self) -> Result<u64, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let ink = self.ink_ready()?;
        trace!("evm_ink_left", self, be!(ink), &[], ink)
    }

    fn msg_reentrant(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let reentrant = self.evm_data().reentrant;
        trace!("msg_reentrant", self, &[], be!(reentrant), reentrant)
    }

    fn msg_sender(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes20(ptr, self.evm_data().msg_sender)?;
        trace!("msg_sender", self, &[], self.evm_data().msg_sender)
    }

    fn msg_value(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes32(ptr, self.evm_data().msg_value)?;
        trace!("msg_value", self, &[], self.evm_data().msg_value)
    }

    fn native_keccak256(&mut self, input: u32, len: u32, output: u32) -> Result<(), Self::Err> {
        self.pay_for_keccak(len)?;

        let preimage = self.read_slice(input, len)?;
        let digest = crypto::keccak(&preimage);
        self.write_bytes32(output, digest.into())?;
        trace!("native_keccak256", self, preimage, digest)
    }

    fn tx_gas_price(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes32(ptr, self.evm_data().tx_gas_price)?;
        trace!("tx_gas_price", self, &[], self.evm_data().tx_gas_price)
    }

    fn tx_ink_price(&mut self) -> Result<u32, Self::Err> {
        self.buy_ink(HOSTIO_INK)?;
        let ink_price = self.pricing().ink_price;
        trace!("tx_ink_price", self, &[], be!(ink_price), ink_price)
    }

    fn tx_origin(&mut self, ptr: u32) -> Result<(), Self::Err> {
        self.buy_ink(HOSTIO_INK + PTR_INK)?;
        self.write_bytes20(ptr, self.evm_data().tx_origin)?;
        trace!("tx_origin", self, &[], self.evm_data().tx_origin)
    }

    fn memory_grow(&mut self, pages: u16) -> Result<(), Self::Err> {
        if pages == 0 {
            self.buy_ink(HOSTIO_INK)?;
            return Ok(());
        }
        let gas_cost = self.evm_api().add_pages(pages);
        self.buy_gas(gas_cost)?;
        trace!("memory_grow", self, be!(pages), &[])
    }

    fn console_log_text(&mut self, ptr: u32, len: u32) -> Result<(), Self::Err> {
        let text = self.read_slice(ptr, len)?;
        self.say(String::from_utf8_lossy(&text));
        trace!("console_log_text", self, text, &[])
    }

    fn console_log<T: Into<Value>>(&mut self, value: T) -> Result<(), Self::Err> {
        let value = value.into();
        self.say(value);
        trace!("console_log", self, [format!("{value}").as_bytes()], &[])
    }

    fn console_tee<T: Into<Value> + Copy>(&mut self, value: T) -> Result<T, Self::Err> {
        self.say(value.into());
        Ok(value)
    }
}
