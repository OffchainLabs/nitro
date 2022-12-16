// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{DefaultFuncMiddleware, Middleware, ModuleMod};
use eyre::Result;
use wasmer::LocalFunctionIndex;

#[derive(Debug)]
pub struct StartMover {
    name: String,
}

impl StartMover {
    pub fn new(name: &str) -> Self {
        let name = name.to_owned();
        Self { name }
    }
}

impl<M: ModuleMod> Middleware<M> for StartMover {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        module.move_start_function(&self.name)
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(DefaultFuncMiddleware)
    }

    fn name(&self) -> &'static str {
        "start mover"
    }
}
