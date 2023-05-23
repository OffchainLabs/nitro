// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    evm::{
        api::{EvmApi, EvmApiMethod, EvmApiStatus},
        user::UserOutcomeKind,
    },
    Bytes20, Bytes32,
};
use eyre::{bail, eyre, Result};
use std::fmt::Debug;

pub struct JsEvmApi<T: JsCallIntoGo> {
    object_ids: Vec<u32>,
    caller: T,
}

pub trait JsCallIntoGo: Send + 'static {
    fn call_go(&mut self, func: u32, args: Vec<ApiValue>) -> Vec<ApiValue>;
}

#[derive(Clone)]
pub struct ApiValue(pub Vec<u8>);

#[derive(Debug)]
enum ApiValueKind {
    U16(u16),
    U32(u32),
    U64(u64),
    Bytes(Bytes),
    Bytes20(Bytes20),
    Bytes32(Bytes32),
    String(String),
    Nil,
}

type Bytes = Vec<u8>;

impl Debug for ApiValue {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let data = &self.0;
        f.write_fmt(format_args!("{}_", data[0]))?;
        f.write_str(&hex::encode(&data[1..]))
    }
}

impl ApiValueKind {
    fn discriminant(&self) -> u8 {
        match self {
            ApiValueKind::U16(_) => 0,
            ApiValueKind::U32(_) => 1,
            ApiValueKind::U64(_) => 2,
            ApiValueKind::Bytes(_) => 3,
            ApiValueKind::Bytes20(_) => 4,
            ApiValueKind::Bytes32(_) => 5,
            ApiValueKind::String(_) => 6,
            ApiValueKind::Nil => 7,
        }
    }
}

impl From<ApiValue> for ApiValueKind {
    fn from(value: ApiValue) -> Self {
        let kind = value.0[0];
        let data = &value.0[1..];
        match kind {
            0 => ApiValueKind::U16(u16::from_be_bytes(data.try_into().unwrap())),
            1 => ApiValueKind::U32(u32::from_be_bytes(data.try_into().unwrap())),
            2 => ApiValueKind::U64(u64::from_be_bytes(data.try_into().unwrap())),
            3 => ApiValueKind::Bytes(data.to_vec()),
            4 => ApiValueKind::Bytes20(data.try_into().unwrap()),
            5 => ApiValueKind::Bytes32(data.try_into().unwrap()),
            6 => ApiValueKind::String(String::from_utf8(data.to_vec()).unwrap()),
            7 => ApiValueKind::Nil,
            _ => unreachable!(),
        }
    }
}

impl From<ApiValueKind> for ApiValue {
    fn from(value: ApiValueKind) -> Self {
        use ApiValueKind::*;
        let mut data = vec![value.discriminant()];
        data.extend(match value {
            U16(x) => x.to_be_bytes().to_vec(),
            U32(x) => x.to_be_bytes().to_vec(),
            U64(x) => x.to_be_bytes().to_vec(),
            Bytes(x) => x,
            Bytes20(x) => x.0.as_ref().to_vec(),
            Bytes32(x) => x.0.as_ref().to_vec(),
            String(x) => x.as_bytes().to_vec(),
            Nil => vec![],
        });
        Self(data)
    }
}

impl From<u16> for ApiValue {
    fn from(value: u16) -> Self {
        ApiValueKind::U16(value).into()
    }
}

impl From<u32> for ApiValue {
    fn from(value: u32) -> Self {
        ApiValueKind::U32(value).into()
    }
}

impl From<u64> for ApiValue {
    fn from(value: u64) -> Self {
        ApiValueKind::U64(value).into()
    }
}

impl From<Bytes> for ApiValue {
    fn from(value: Bytes) -> Self {
        ApiValueKind::Bytes(value).into()
    }
}

impl From<Bytes20> for ApiValue {
    fn from(value: Bytes20) -> Self {
        ApiValueKind::Bytes20(value).into()
    }
}

impl From<Bytes32> for ApiValue {
    fn from(value: Bytes32) -> Self {
        ApiValueKind::Bytes32(value).into()
    }
}

impl From<String> for ApiValue {
    fn from(value: String) -> Self {
        ApiValueKind::String(value).into()
    }
}

impl ApiValueKind {
    fn assert_u16(self) -> u16 {
        match self {
            ApiValueKind::U16(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_u32(self) -> u32 {
        match self {
            ApiValueKind::U32(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_u64(self) -> u64 {
        match self {
            ApiValueKind::U64(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_bytes(self) -> Bytes {
        match self {
            ApiValueKind::Bytes(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_bytes32(self) -> Bytes32 {
        match self {
            ApiValueKind::Bytes32(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_status(self) -> UserOutcomeKind {
        match self {
            ApiValueKind::Nil => EvmApiStatus::Success.into(),
            ApiValueKind::String(_) => EvmApiStatus::Failure.into(),
            x => panic!("wrong type {x:?}"),
        }
    }
}

impl<T: JsCallIntoGo> JsEvmApi<T> {
    pub fn new(ids: Vec<u8>, caller: T) -> Self {
        let mut object_ids = vec![];
        for i in (0..ids.len()).step_by(4) {
            let slice = &ids[i..(i + 4)];
            let value = u32::from_be_bytes(slice.try_into().unwrap());
            object_ids.push(value);
        }
        Self { object_ids, caller }
    }

    fn call(&mut self, func: EvmApiMethod, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let func_id = self.object_ids[func as usize];
        self.caller.call_go(func_id, args)
    }
}

macro_rules! call {
    ($self:expr, $num:expr, $func:ident $(,$args:expr)*) => {{
        let outs = $self.call(EvmApiMethod::$func, vec![$($args.into()),*]);
        let x: [ApiValue; $num] = outs.try_into().unwrap();
        let x: [ApiValueKind; $num] = x.map(Into::into);
        x
    }};
}

impl<T: JsCallIntoGo> EvmApi for JsEvmApi<T> {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let [value, cost] = call!(self, 2, GetBytes32, key);
        (value.assert_bytes32(), cost.assert_u64())
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let [out] = call!(self, 1, SetBytes32, key, value);
        match out {
            ApiValueKind::U64(value) => Ok(value),
            ApiValueKind::String(err) => bail!(err),
            _ => unreachable!(),
        }
    }

    fn contract_call(
        &mut self,
        contract: Bytes20,
        input: Bytes,
        gas: u64,
        value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        let [len, cost, status] = call!(self, 3, ContractCall, contract, input, gas, value);
        (len.assert_u32(), cost.assert_u64(), status.assert_status())
    }

    fn delegate_call(
        &mut self,
        contract: Bytes20,
        input: Bytes,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        let [len, cost, status] = call!(self, 3, DelegateCall, contract, input, gas);
        (len.assert_u32(), cost.assert_u64(), status.assert_status())
    }

    fn static_call(
        &mut self,
        contract: Bytes20,
        input: Bytes,
        gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        let [len, cost, status] = call!(self, 3, StaticCall, contract, input, gas);
        (len.assert_u32(), cost.assert_u64(), status.assert_status())
    }

    fn create1(
        &mut self,
        code: Bytes,
        endowment: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        let [result, len, cost] = call!(self, 3, Create1, code, endowment, gas);
        let result = match result {
            ApiValueKind::Bytes20(account) => Ok(account),
            ApiValueKind::String(err) => Err(eyre!(err)),
            _ => unreachable!(),
        };
        (result, len.assert_u32(), cost.assert_u64())
    }

    fn create2(
        &mut self,
        code: Bytes,
        endowment: Bytes32,
        salt: Bytes32,
        gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        let [result, len, cost] = call!(self, 3, Create2, code, endowment, salt, gas);
        let result = match result {
            ApiValueKind::Bytes20(account) => Ok(account),
            ApiValueKind::String(err) => Err(eyre!(err)),
            _ => unreachable!(),
        };
        (result, len.assert_u32(), cost.assert_u64())
    }

    fn get_return_data(&mut self) -> Bytes {
        let [data] = call!(self, 1, GetReturnData);
        data.assert_bytes()
    }

    fn emit_log(&mut self, data: Bytes, topics: u32) -> Result<()> {
        let [out] = call!(self, 1, EmitLog, data, topics);
        match out {
            ApiValueKind::Nil => Ok(()),
            ApiValueKind::String(err) => bail!(err),
            _ => unreachable!(),
        }
    }

    fn account_balance(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let [value, cost] = call!(self, 2, AccountBalance, address);
        (value.assert_bytes32(), cost.assert_u64())
    }

    fn account_codehash(&mut self, address: Bytes20) -> (Bytes32, u64) {
        let [value, cost] = call!(self, 2, AccountCodeHash, address);
        (value.assert_bytes32(), cost.assert_u64())
    }

    fn evm_blockhash(&mut self, num: Bytes32) -> Bytes32 {
        let [value] = call!(self, 1, EvmBlockHash, num);
        value.assert_bytes32()
    }

    fn add_pages(&mut self, pages: u16) -> (u16, u16) {
        let [open, ever] = call!(self, 2, AddPages, pages);
        (open.assert_u16(), ever.assert_u16())
    }
}
