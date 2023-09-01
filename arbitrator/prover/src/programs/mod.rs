// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    binary::{ExportKind, WasmBinary},
    memory::MemoryType,
    value::{FunctionType as ArbFunctionType, Value},
};
use arbutil::Color;
use eyre::{bail, eyre, Report, Result};
use fnv::FnvHashMap as HashMap;
use std::fmt::Debug;
use wasmer_types::{
    entity::EntityRef, FunctionIndex, GlobalIndex, GlobalInit, ImportIndex, LocalFunctionIndex,
    SignatureIndex, Type,
};
use wasmparser::{Operator, Type as WpType};

#[cfg(feature = "native")]
use {
    super::value,
    std::marker::PhantomData,
    wasmer::{
        ExportIndex, FunctionMiddleware, GlobalType, MiddlewareError, ModuleMiddleware, Mutability,
    },
    wasmer_types::{MemoryIndex, ModuleInfo},
};

pub mod config;
pub mod counter;
pub mod depth;
pub mod dynamic;
pub mod heap;
pub mod memory;
pub mod meter;
pub mod prelude;
pub mod start;

pub const STYLUS_ENTRY_POINT: &str = "user_entrypoint";

pub trait ModuleMod {
    fn add_global(&mut self, name: &str, ty: Type, init: GlobalInit) -> Result<GlobalIndex>;
    fn get_global(&mut self, name: &str) -> Result<GlobalIndex>;
    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType>;
    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType>;
    fn all_functions(&self) -> Result<HashMap<FunctionIndex, ArbFunctionType>>;
    fn all_signatures(&self) -> Result<HashMap<SignatureIndex, ArbFunctionType>>;
    fn get_import(&self, module: &str, name: &str) -> Result<ImportIndex>;
    fn move_start_function(&mut self, name: &str) -> Result<()>;
    fn memory_info(&self) -> Result<MemoryType>;
}

pub trait Middleware<M: ModuleMod> {
    type FM<'a>: FuncMiddleware<'a> + Debug;

    fn update_module(&self, module: &mut M) -> Result<()>; // not mutable due to wasmer
    fn instrument<'a>(&self, func_index: LocalFunctionIndex) -> Result<Self::FM<'a>>;
    fn name(&self) -> &'static str;
}

pub trait FuncMiddleware<'a> {
    /// Provide info on the function's locals. This is called before feed.
    fn locals_info(&mut self, _locals: &[WpType]) {}

    /// Processes the given operator.
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>;

    /// The name of the middleware
    fn name(&self) -> &'static str;
}

#[derive(Debug)]
pub struct DefaultFuncMiddleware;

impl<'a> FuncMiddleware<'a> for DefaultFuncMiddleware {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        out.extend([op]);
        Ok(())
    }

    fn name(&self) -> &'static str {
        "default middleware"
    }
}

/// This wrapper exists to impl wasmer's `ModuleMiddleware` generically.
/// We can't use `T` directly since we don't define `ModuleMiddleware`,
/// and we need `M` to be part of the type.
#[cfg(feature = "native")]
#[derive(Debug)]
pub struct MiddlewareWrapper<T, M>(pub T, PhantomData<M>)
where
    T: Middleware<M> + Debug + Send + Sync,
    M: ModuleMod;

#[cfg(feature = "native")]
impl<T, M> MiddlewareWrapper<T, M>
where
    T: Middleware<M> + Debug + Send + Sync,
    M: ModuleMod,
{
    pub fn new(middleware: T) -> Self {
        Self(middleware, PhantomData)
    }
}

#[cfg(feature = "native")]
impl<T> ModuleMiddleware for MiddlewareWrapper<T, ModuleInfo>
where
    T: Middleware<ModuleInfo> + Debug + Send + Sync + 'static,
{
    fn transform_module_info(&self, module: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let error = |err| MiddlewareError::new(self.0.name().red(), format!("{:?}", err));
        self.0.update_module(module).map_err(error)
    }

    fn generate_function_middleware<'a>(
        &self,
        local_function_index: LocalFunctionIndex,
    ) -> Box<dyn wasmer::FunctionMiddleware<'a> + 'a> {
        let worker = self.0.instrument(local_function_index).unwrap();
        Box::new(FuncMiddlewareWrapper(worker, PhantomData))
    }
}

/// This wrapper exists to impl wasmer's `FunctionMiddleware` generically.
/// The logic is analogous to that of `ModuleMiddleware`, except this time
/// we need a phantom marker to parameterize by `T`'s reference's lifetime.
#[cfg(feature = "native")]
#[derive(Debug)]
pub struct FuncMiddlewareWrapper<'a, T: 'a>(T, PhantomData<&'a T>)
where
    T: FuncMiddleware<'a> + Debug;

#[cfg(feature = "native")]
impl<'a, T> FunctionMiddleware<'a> for FuncMiddlewareWrapper<'a, T>
where
    T: FuncMiddleware<'a> + Debug,
{
    fn locals_info(&mut self, locals: &[WpType]) {
        self.0.locals_info(locals);
    }

    fn feed(
        &mut self,
        op: Operator<'a>,
        out: &mut wasmer::MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        let name = self.0.name().red();
        let error = |err| MiddlewareError::new(name, format!("{:?}", err));
        self.0.feed(op, out).map_err(error)
    }
}

#[cfg(feature = "native")]
impl ModuleMod for ModuleInfo {
    fn add_global(&mut self, name: &str, ty: Type, init: GlobalInit) -> Result<GlobalIndex> {
        let global_type = GlobalType::new(ty, Mutability::Var);
        let name = name.to_owned();
        if self.exports.contains_key(&name) {
            bail!("wasm already contains {}", name.red())
        }
        let index = self.globals.push(global_type);
        self.exports.insert(name, ExportIndex::Global(index));
        self.global_initializers.push(init);
        Ok(index)
    }

    fn get_global(&mut self, name: &str) -> Result<GlobalIndex> {
        let Some(ExportIndex::Global(global)) = self.exports.get(name) else {
            bail!("missing global {}", name.red())
        };
        Ok(*global)
    }

    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType> {
        let error = Report::msg(format!("missing signature {}", sig.as_u32().red()));
        let ty = self.signatures.get(sig).cloned().ok_or(error)?;
        let ty = value::parser_func_type(ty);
        ty.try_into()
    }

    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType> {
        let index = func.as_u32();
        match self.functions.get(func) {
            Some(sig) => self.get_signature(*sig),
            None => match self.function_names.get(&func) {
                Some(name) => bail!("missing func {} @ index {}", name.red(), index.red()),
                None => bail!("missing func @ index {}", index.red()),
            },
        }
    }

    fn all_functions(&self) -> Result<HashMap<FunctionIndex, ArbFunctionType>> {
        let mut funcs = HashMap::default();
        for (func, sig) in &self.functions {
            let ty = self.get_signature(*sig)?;
            funcs.insert(func, ty);
        }
        Ok(funcs)
    }

    fn all_signatures(&self) -> Result<HashMap<SignatureIndex, ArbFunctionType>> {
        let mut signatures = HashMap::default();
        for (index, _) in &self.signatures {
            let ty = self.get_signature(index)?;
            signatures.insert(index, ty);
        }
        Ok(signatures)
    }

    fn get_import(&self, module: &str, name: &str) -> Result<ImportIndex> {
        self.imports
            .iter()
            .find(|(k, _)| k.module == module && k.field == name)
            .map(|(_, v)| v.clone())
            .ok_or_else(|| eyre!("missing import {}", name.red()))
    }

    fn move_start_function(&mut self, name: &str) -> Result<()> {
        if let Some(prior) = self.exports.get(name) {
            bail!("function {} already exists @ index {:?}", name.red(), prior)
        }

        if let Some(start) = self.start_function.take() {
            let export = ExportIndex::Function(start);
            self.exports.insert(name.to_owned(), export);
            self.function_names.insert(start, name.to_owned());
        }
        Ok(())
    }

    fn memory_info(&self) -> Result<MemoryType> {
        if self.memories.is_empty() {
            bail!("missing memory export with name {}", "memory".red());
        }
        if self.memories.len() > 1 {
            bail!("only one memory is allowed");
        }
        if self.exports.get("memory") != Some(&ExportIndex::Memory(MemoryIndex::from_u32(0))) {
            bail!("missing memory with export name {}", "memory".red());
        }
        Ok(self.memories.last().unwrap().into())
    }
}

impl<'a> ModuleMod for WasmBinary<'a> {
    fn add_global(&mut self, name: &str, _ty: Type, init: GlobalInit) -> Result<GlobalIndex> {
        let global = match init {
            GlobalInit::I32Const(x) => Value::I32(x as u32),
            GlobalInit::I64Const(x) => Value::I64(x as u64),
            GlobalInit::F32Const(x) => Value::F32(x),
            GlobalInit::F64Const(x) => Value::F64(x),
            ty => bail!("cannot add global of type {:?}", ty),
        };
        if self.exports.contains_key(name) {
            bail!("wasm already contains {}", name.red())
        }
        let name = name.to_owned();
        let index = self.globals.len() as u32;
        self.exports.insert(name, (index, ExportKind::Global));
        self.globals.push(global);
        Ok(GlobalIndex::from_u32(index))
    }

    fn get_global(&mut self, name: &str) -> Result<GlobalIndex> {
        let Some((global, ExportKind::Global)) = self.exports.get(name) else {
            bail!("missing global {}", name.red())
        };
        Ok(GlobalIndex::from_u32(*global))
    }

    fn get_signature(&self, sig: SignatureIndex) -> Result<ArbFunctionType> {
        let index = sig.as_u32() as usize;
        let error = Report::msg(format!("missing signature {}", index.red()));
        self.types.get(index).cloned().ok_or(error)
    }

    fn get_function(&self, func: FunctionIndex) -> Result<ArbFunctionType> {
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
                Some(name) => bail!("missing func {} @ index {}", name.red(), func.red()),
                None => bail!("missing func @ index {}", func.red()),
            },
        }
    }

    fn all_functions(&self) -> Result<HashMap<FunctionIndex, ArbFunctionType>> {
        let mut funcs = HashMap::default();
        let mut index = 0;
        for import in &self.imports {
            let ty = self.get_signature(SignatureIndex::from_u32(import.offset))?;
            funcs.insert(FunctionIndex::new(index), ty);
            index += 1;
        }
        for sig in &self.functions {
            let ty = self.get_signature(SignatureIndex::from_u32(*sig))?;
            funcs.insert(FunctionIndex::new(index), ty);
            index += 1;
        }
        Ok(funcs)
    }

    fn all_signatures(&self) -> Result<HashMap<SignatureIndex, ArbFunctionType>> {
        let mut signatures = HashMap::default();
        for (index, ty) in self.types.iter().enumerate() {
            let sig = SignatureIndex::new(index);
            signatures.insert(sig, ty.clone());
        }
        Ok(signatures)
    }

    fn get_import(&self, module: &str, name: &str) -> Result<ImportIndex> {
        self.imports
            .iter()
            .position(|x| x.module == module && x.name == Some(name))
            .map(|x| ImportIndex::Function(FunctionIndex::from_u32(x as u32)))
            .ok_or_else(|| eyre!("missing import {}", name.red()))
    }

    fn move_start_function(&mut self, name: &str) -> Result<()> {
        if let Some(prior) = self.exports.get(name) {
            bail!("function {} already exists @ index {:?}", name.red(), prior)
        }

        if let Some(start) = self.start.take() {
            let name = name.to_owned();
            self.exports.insert(name.clone(), (start, ExportKind::Func));
            self.names.functions.insert(start, name);
        }
        Ok(())
    }

    fn memory_info(&self) -> Result<MemoryType> {
        if self.memories.is_empty() {
            bail!("missing memory export with name {}", "memory".red());
        }
        if self.memories.len() > 1 {
            bail!("only one memory is allowed");
        }
        if self.exports.get("memory") != Some(&(0, ExportKind::Memory)) {
            bail!("missing memory with export name {}", "memory".red());
        }
        self.memories.last().unwrap().try_into()
    }
}

#[derive(Clone, Copy, Debug)]
pub struct StylusData {
    pub ink_left: GlobalIndex,
    pub ink_status: GlobalIndex,
    pub depth_left: GlobalIndex,
    pub footprint: u16,
}

impl StylusData {
    pub fn global_offsets(&self) -> (u64, u64, u64) {
        (
            self.ink_left.as_u32() as u64,
            self.ink_status.as_u32() as u64,
            self.depth_left.as_u32() as u64,
        )
    }
}
