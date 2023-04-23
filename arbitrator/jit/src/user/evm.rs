// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::Color;
use eyre::{bail, Result};
use prover::{
    programs::run::UserOutcomeKind,
    utils::{Bytes20, Bytes32},
};
use std::{
    fmt::Debug,
    sync::mpsc::{self, SyncSender},
};
use stylus::EvmApi;

pub(super) struct JitApi {
    object_ids: Vec<u32>,
    parent: SyncSender<EvmMsg>,
}

pub(super) enum EvmMsg {
    Call(u32, Vec<ApiValue>, SyncSender<Vec<ApiValue>>),
    Panic(String),
    Done,
}

#[derive(Clone, Debug)]
pub(super) struct ApiValue(pub Vec<u8>);

#[derive(Debug)]
enum ApiValueKind {
    U64(u64),
    Bytes32(Bytes32),
    String(String),
    Nil,
}

impl ApiValueKind {
    fn discriminant(&self) -> u8 {
        match self {
            ApiValueKind::U64(_) => 0,
            ApiValueKind::Bytes32(_) => 1,
            ApiValueKind::String(_) => 2,
            ApiValueKind::Nil => 3,
        }
    }
}

impl From<ApiValue> for ApiValueKind {
    fn from(value: ApiValue) -> Self {
        let kind = value.0[0];
        let data = &value.0[1..];
        match kind {
            0 => ApiValueKind::U64(u64::from_be_bytes(data.try_into().unwrap())),
            1 => ApiValueKind::Bytes32(data.try_into().unwrap()),
            2 => ApiValueKind::String(String::from_utf8(data.to_vec()).unwrap()),
            3 => ApiValueKind::Nil,
            _ => unreachable!(),
        }
    }
}

impl From<ApiValueKind> for ApiValue {
    fn from(value: ApiValueKind) -> Self {
        use ApiValueKind::*;
        let mut data = vec![value.discriminant()];
        data.extend(match value {
            U64(x) => x.to_be_bytes().to_vec(),
            Bytes32(x) => x.0.as_ref().to_vec(),
            String(x) => x.as_bytes().to_vec(),
            Nil => vec![],
        });
        Self(data)
    }
}

impl From<u64> for ApiValue {
    fn from(value: u64) -> Self {
        ApiValueKind::U64(value).into()
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
    fn assert_u64(self) -> u64 {
        match self {
            ApiValueKind::U64(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }

    fn assert_bytes32(self) -> Bytes32 {
        match self {
            ApiValueKind::Bytes32(value) => value,
            x => panic!("wrong type {x:?}"),
        }
    }
}

impl JitApi {
    pub fn new(ids: Vec<u8>, parent: SyncSender<EvmMsg>) -> Self {
        let mut object_ids = vec![];
        for i in 0..2 {
            let start = i * 4;
            let slice = &ids[start..(start + 4)];
            let value = u32::from_be_bytes(slice.try_into().unwrap());
            println!("Func id {}", value.pink());
            object_ids.push(value);
        }
        Self { object_ids, parent }
    }

    fn exec(&mut self, func: usize, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let (tx, rx) = mpsc::sync_channel(0);
        let func = self.object_ids[func];
        let msg = EvmMsg::Call(func, args, tx);
        self.parent.send(msg).unwrap();
        rx.recv().unwrap()
    }
}

macro_rules! cast {
    ($num:expr, $outs:expr) => {{
        let x: [ApiValue; $num] = $outs.try_into().unwrap();
        let x: [ApiValueKind; $num] = x.map(Into::into);
        x
    }};
}

impl EvmApi for JitApi {
    fn get_bytes32(&mut self, key: Bytes32) -> (Bytes32, u64) {
        let outs = self.exec(0, vec![key.into()]);
        let [value, cost] = cast!(2, outs);
        (value.assert_bytes32(), cost.assert_u64())
    }

    fn set_bytes32(&mut self, key: Bytes32, value: Bytes32) -> Result<u64> {
        let outs = self.exec(1, vec![key.into(), value.into()]);
        let [out] = cast!(1, outs);
        match out {
            ApiValueKind::U64(value) => Ok(value),
            ApiValueKind::String(err) => bail!(err),
            _ => unreachable!(),
        }
    }

    fn contract_call(
        &mut self,
        _contract: Bytes20,
        _input: Vec<u8>,
        _gas: u64,
        _value: Bytes32,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!()
    }

    fn delegate_call(
        &mut self,
        _contract: Bytes20,
        _input: Vec<u8>,
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!()
    }

    fn static_call(
        &mut self,
        _contract: Bytes20,
        _input: Vec<u8>,
        _gas: u64,
    ) -> (u32, u64, UserOutcomeKind) {
        todo!()
    }

    fn create1(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        todo!()
    }

    fn create2(
        &mut self,
        _code: Vec<u8>,
        _endowment: Bytes32,
        _salt: Bytes32,
        _gas: u64,
    ) -> (Result<Bytes20>, u32, u64) {
        todo!()
    }

    fn load_return_data(&mut self) -> Vec<u8> {
        todo!()
    }

    fn emit_log(&mut self, _data: Vec<u8>, _topics: usize) -> Result<()> {
        todo!()
    }
}
