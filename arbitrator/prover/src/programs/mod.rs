// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{binary::WasmBinary, value::FunctionType as ArbFunctionType};

use arbutil::Color;
use wasmer_types::{FunctionIndex, SignatureIndex};

pub trait ModuleMod {
    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType, String>;
    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType, String>;
}

impl<'a> ModuleMod for WasmBinary<'a> {
    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType, String> {
        let index = sig.as_u32() as usize;
        let error = || format!("missing signature {}", index.red());
        let ty = self.types.get(index).ok_or_else(error)?;
        ty.clone().try_into().map_err(|_| error())
    }

    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType, String> {
        let mut index = func.as_u32() as usize;
        let sig;

        if index < self.imports.len() {
            sig = self.imports.get(index).map(|x| &x.offset);
        } else {
            index -= self.imports.len();
            sig = self.functions.get(index);
        }

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
