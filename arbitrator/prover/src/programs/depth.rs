// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{FuncMiddleware, Middleware, ModuleMod};
use crate::value::FunctionType;

use arbutil::Color;
use eyre::{bail, Result};
use fnv::FnvHashMap as HashMap;
use parking_lot::Mutex;
use std::sync::Arc;
use wasmer::wasmparser::{Operator, Type as WpType, TypeOrFuncType};
use wasmer_types::{
    FunctionIndex, GlobalIndex, GlobalInit, LocalFunctionIndex, SignatureIndex, Type,
};

const POLYGLOT_STACK_LEFT: &str = "polyglot_stack_left";

/// This middleware ensures stack overflows are deterministic across different compilers and targets.
/// The internal notion of "stack space left" that makes this possible is strictly smaller than that of
/// the real stack space consumed on any target platform and is formed by inspecting the contents of each
/// function's frame.
/// Setting a limit smaller than that of any native platform's ensures stack overflows will have the same,
/// logical effect rather than actually exhausting the space provided by the OS.
#[derive(Debug)]
pub struct DepthChecker {
    /// The amount of stack space left
    pub global: Mutex<Option<GlobalIndex>>,
    /// The maximum size of the stack, measured in words
    limit: u32,
    /// The function types of the module being instrumented
    funcs: Mutex<Arc<HashMap<FunctionIndex, FunctionType>>>,
    /// The types of the module being instrumented
    sigs: Mutex<Arc<HashMap<SignatureIndex, FunctionType>>>,
}

impl DepthChecker {
    pub fn new(limit: u32) -> Self {
        Self {
            global: Mutex::new(None),
            limit,
            funcs: Mutex::new(Arc::new(HashMap::default())),
            sigs: Mutex::new(Arc::new(HashMap::default())),
        }
    }
}

impl<M: ModuleMod> Middleware<M> for DepthChecker {
    type FM<'a> = FuncDepthChecker<'a>;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let limit = GlobalInit::I32Const(self.limit as i32);
        let space = module.add_global(POLYGLOT_STACK_LEFT, Type::I32, limit)?;
        *self.global.lock() = Some(space);
        *self.funcs.lock() = Arc::new(module.all_functions()?);
        *self.sigs.lock() = Arc::new(module.all_signatures()?);
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        let global = self.global.lock().unwrap();
        let funcs = self.funcs.lock().clone();
        let sigs = self.sigs.lock().clone();
        let limit = self.limit;
        Ok(FuncDepthChecker::new(global, funcs, sigs, limit))
    }

    fn name(&self) -> &'static str {
        "depth checker"
    }
}

#[derive(Debug)]
pub struct FuncDepthChecker<'a> {
    /// The amount of stack space left
    global: GlobalIndex,
    /// The function types in this function's module
    funcs: Arc<HashMap<FunctionIndex, FunctionType>>,
    /// All the types in this function's modules
    sigs: Arc<HashMap<SignatureIndex, FunctionType>>,
    /// The maximum size of the stack, measured in words
    limit: u32,
    scopes: isize,
    code: Vec<Operator<'a>>,
    done: bool,
}

impl<'a> FuncDepthChecker<'a> {
    fn new(
        global: GlobalIndex,
        funcs: Arc<HashMap<FunctionIndex, FunctionType>>,
        sigs: Arc<HashMap<SignatureIndex, FunctionType>>,
        limit: u32,
    ) -> Self {
        Self {
            global,
            funcs,
            sigs,
            limit,
            scopes: 1, // a function starts with an open scope
            code: vec![],
            done: false,
        }
    }
}

impl<'a> FuncMiddleware<'a> for FuncDepthChecker<'a> {
    fn feed<O>(&mut self, op: wasmer::wasmparser::Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<wasmer::wasmparser::Operator<'a>>,
    {
        use Operator::*;

        // Knowing when the feed ends requires detecting the final instruction, which is
        // guaranteed to be an "End" opcode closing out function's initial opening scope.
        if self.done {
            bail!("finalized too soon");
        }

        let scopes = &mut self.scopes;
        match op {
            Block { .. } | Loop { .. } | If { .. } => *scopes += 1,
            End => *scopes -= 1,
            _ => {}
        }
        if *scopes < 0 {
            bail!("malformed scoping detected");
        }

        let last = *scopes == 0 && matches!(op, End); // true when the feed ends
        self.code.push(op);
        if !last {
            return Ok(());
        }

        // We've reached the final instruction and can instrument the function as follows:
        //   - When entering, check that the stack has sufficient space and deduct the amount used
        //   - When returning, credit back the amount used as execution is returning to the caller

        let mut code = std::mem::replace(&mut self.code, vec![]);
        let size = 1;
        let global_index = self.global.as_u32();
        let max_frame_size = self.limit / 4;

        if size > max_frame_size {
            let limit = max_frame_size.red();
            bail!("frame too large: {} > {}-word limit", size.red(), limit);
        }

        out.extend(vec![
            // if space <= size => panic with depth = 0
            GlobalGet { global_index },
            I32Const { value: size as i32 },
            I32LeU,
            If {
                ty: TypeOrFuncType::Type(WpType::EmptyBlockType),
            },
            I32Const { value: 0 },
            GlobalSet { global_index },
            Unreachable,
            End,
            // space -= size
            GlobalGet { global_index },
            I32Const { value: size as i32 },
            I32Sub,
            GlobalSet { global_index },
        ]);

        let reclaim = |out: &mut O| {
            out.extend(vec![
                // space += size
                GlobalGet { global_index },
                I32Const { value: size as i32 },
                I32Add,
                GlobalSet { global_index },
            ])
        };

        // add an extraneous return instruction to the end to match Arbitrator
        let last = code.pop().unwrap();
        code.push(Return);
        code.push(last);

        for op in code {
            let exit = matches!(op, Return);
            if exit {
                reclaim(out);
            }
            out.extend(vec![op]);
        }

        self.done = true;
        Ok(())
    }

    fn name(&self) -> &'static str {
        "depth checker"
    }
}
