// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    binary::ExportKind,
    programs::{DefaultFuncMiddleware, Middleware, ModuleMod, STYLUS_ENTRY_POINT},
};
use eyre::{bail, Result};
use fnv::FnvHashMap as HashMap;
use lazy_static::lazy_static;
use wasmer_types::LocalFunctionIndex;

#[cfg(feature = "native")]
use wasmer::TypedFunction;

lazy_static! {
    /// Lists the exports a user program map have
    static ref EXPORT_WHITELIST: HashMap<&'static str, ExportKind> = {
        let mut map = HashMap::default();
        map.insert(STYLUS_ENTRY_POINT, ExportKind::Func);
        map.insert(StartMover::NAME, ExportKind::Func);
        map.insert("memory", ExportKind::Memory);
        map
    };
}

#[derive(Debug)]
pub struct StartMover {
    /// Whether to keep offchain information.
    debug: bool,
}

impl StartMover {
    pub const NAME: &'static str = "stylus_start";

    pub fn new(debug: bool) -> Self {
        Self { debug }
    }
}

impl<M: ModuleMod> Middleware<M> for StartMover {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let had_start = module.move_start_function(Self::NAME)?;
        if had_start && !self.debug {
            bail!("start functions not allowed");
        }
        if !self.debug {
            module.drop_exports_and_names(&EXPORT_WHITELIST);
        }
        Ok(())
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
    fn get_start(&self) -> Result<TypedFunction<(), ()>>;
}
