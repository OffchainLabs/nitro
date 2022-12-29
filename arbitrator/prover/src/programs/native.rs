// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![cfg(feature = "native")]

use arbutil::Color;
use eyre::{bail, ErrReport, Result};
use std::{
    fmt::Debug,
    ops::{Deref, DerefMut},
};
use wasmer::{AsStoreMut, Instance, Store, Value as WasmerValue};

pub struct NativeInstance {
    pub instance: Instance,
    pub store: Store,
}

impl NativeInstance {
    pub fn new(instance: Instance, store: Store) -> Self {
        Self { instance, store }
    }
}

impl Deref for NativeInstance {
    type Target = Instance;

    fn deref(&self) -> &Self::Target {
        &self.instance
    }
}

impl DerefMut for NativeInstance {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.instance
    }
}

pub trait GlobalMod {
    fn get_global<T>(&mut self, name: &str) -> Result<T>
    where
        T: TryFrom<WasmerValue>,
        T::Error: Debug;

    fn set_global<T>(&mut self, name: &str, value: T) -> Result<()>
    where
        T: Into<WasmerValue>;
}

impl GlobalMod for NativeInstance {
    fn get_global<T>(&mut self, name: &str) -> Result<T>
    where
        T: TryFrom<WasmerValue>,
        T::Error: Debug,
    {
        let store = &mut self.store.as_store_mut();
        let Ok(global) = self.instance.exports.get_global(name) else {
            bail!("global {} does not exist", name.red())
        };
        let ty = global.get(store);

        let error = || format!("global {} has the wrong type", name.red());
        ty.try_into().map_err(|_| ErrReport::msg(error()))
    }

    fn set_global<T>(&mut self, name: &str, value: T) -> Result<()>
    where
        T: Into<WasmerValue>,
    {
        let store = &mut self.store.as_store_mut();
        let Ok(global) = self.instance.exports.get_global(name) else {
            bail!("global {} does not exist", name.red())
        };
        global.set(store, value.into()).map_err(ErrReport::msg)
    }
}
