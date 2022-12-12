// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{binary::WasmBinary, value::FunctionType as ArbFunctionType};

use arbutil::Color;
use wasmer_types::{FunctionIndex, SignatureIndex};

pub mod config;

pub trait ModuleMod {
    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType, String>;
    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType, String>;
}

impl<'a> ModuleMod for WasmBinary<'a> {
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
