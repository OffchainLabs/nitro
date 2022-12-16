// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{DefaultFuncMiddleware, Middleware, ModuleMod};
use eyre::{ErrReport, Result};
use wasmer::{Instance, LocalFunctionIndex, Store, TypedFunction};

const POLYGLOT_START: &str = "polyglot_start";

#[derive(Debug, Default)]
pub struct StartMover {}

impl<M: ModuleMod> Middleware<M> for StartMover {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        module.move_start_function(POLYGLOT_START)
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(DefaultFuncMiddleware)
    }

    fn name(&self) -> &'static str {
        "start mover"
    }
}

pub trait StartlessMachine {
    fn get_start(&self, store: &Store) -> Result<TypedFunction<(), ()>>;
}

impl StartlessMachine for Instance {
    fn get_start(&self, store: &Store) -> Result<TypedFunction<(), ()>> {
        self.exports
            .get_typed_function(store, POLYGLOT_START)
            .map_err(|err| ErrReport::new(err))
    }
}
