// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{DefaultFuncMiddleware, Middleware, ModuleMod};
use eyre::Result;
use wasmer_types::LocalFunctionIndex;

#[cfg(feature = "native")]
use {
    eyre::ErrReport,
    wasmer::{Instance, Store, TypedFunction},
};

const STYLUS_START: &str = "stylus_start";

#[derive(Debug, Default)]
pub struct StartMover {}

impl<M: ModuleMod> Middleware<M> for StartMover {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        module.move_start_function(STYLUS_START)
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(DefaultFuncMiddleware)
    }

    fn name(&self) -> &'static str {
        "start mover"
    }
}

#[cfg(feature = "native")]
pub trait StartlessMachine {
    fn get_start(&self, store: &Store) -> Result<TypedFunction<(), ()>>;
}

#[cfg(feature = "native")]
impl StartlessMachine for Instance {
    fn get_start(&self, store: &Store) -> Result<TypedFunction<(), ()>> {
        self.exports
            .get_typed_function(store, STYLUS_START)
            .map_err(ErrReport::new)
    }
}
