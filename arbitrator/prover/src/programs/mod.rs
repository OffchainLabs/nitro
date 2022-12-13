// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    binary::{ExportKind, WasmBinary},
    value::{FunctionType as ArbFunctionType, Value},
};

use arbutil::Color;
use std::fmt::Debug;
use wasmer::{GlobalInit, Instance, Store, Value as WasmerValue};
use wasmer_types::{FunctionIndex, GlobalIndex, SignatureIndex, Type};

pub mod config;
pub mod meter;

pub trait ModuleMod {
    fn add_global(&mut self, name: &str, ty: Type, init: GlobalInit) -> GlobalIndex;
    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType, String>;
    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType, String>;
}

impl<'a> ModuleMod for WasmBinary<'a> {
    fn add_global(&mut self, name: &str, ty: Type, init: GlobalInit) -> GlobalIndex {
        let global = match init {
            GlobalInit::I32Const(x) => Value::I32(x as u32),
            GlobalInit::I64Const(x) => Value::I64(x as u64),
            GlobalInit::F32Const(x) => Value::F32(x),
            GlobalInit::F64Const(x) => Value::F64(x),
            _ => panic!("cannot add global of type {}", ty),
        };

        let name = name.to_owned();
        let index = self.globals.len() as u32;
        self.exports.insert((name, ExportKind::Global), index);
        self.globals.push(global);
        GlobalIndex::from_u32(index)
    }

    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType, String> {
        let index = sig.as_u32() as usize;
        self.types
            .get(index)
            .cloned()
            .ok_or(format!("missing signature {}", index.red()))
    }

    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType, String> {
        let mut index = func.as_u32() as usize;

        let sig = if index < self.imports.len() {
            self.imports.get(index).map(|x| &x.offset)
        } else {
            index -= self.imports.len();
            self.functions.get(index)
        };

        let func = func.as_u32();
        match sig {
            Some(sig) => self.get_signature(SignatureIndex::from_u32(*sig)),
            None => match self.names.functions.get(&func) {
                Some(name) => Err(format!(
                    "missing func {} @ index {}",
                    name.red(),
                    func.red()
                )),
                None => Err(format!("missing func @ index {}", func.red())),
            },
        }
    }
}

pub trait GlobalMod {
    fn get_global<T>(&self, store: &mut Store, name: &str) -> T
    where
        T: TryFrom<WasmerValue>,
        T::Error: Debug;

    fn set_global<T>(&mut self, store: &mut Store, name: &str, value: T)
    where
        T: Into<WasmerValue>;
}

impl GlobalMod for Instance {
    fn get_global<T>(&self, store: &mut Store, name: &str) -> T
    where
        T: TryFrom<WasmerValue>,
        T::Error: Debug,
    {
        let error = format!("global {} does not exist", name.red());
        let global = self.exports.get_global(name).expect(&error);
        global.get(store).try_into().expect("wrong type")
    }

    fn set_global<T>(&mut self, store: &mut Store, name: &str, value: T)
    where
        T: Into<WasmerValue>,
    {
        let error = format!("global {} does not exist", name.red());
        let global = self.exports.get_global(name).expect(&error);
        global.set(store, value.into()).unwrap();
    }
}
