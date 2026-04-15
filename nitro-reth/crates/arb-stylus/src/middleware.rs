use std::{collections::HashMap, sync::RwLock};

use wasmer_compiler::{FunctionMiddleware, MiddlewareReaderState, ModuleMiddleware};
use wasmer_types::{
    ExportIndex, FunctionIndex, FunctionType, GlobalIndex, GlobalInit, ImportIndex,
    LocalFunctionIndex, MiddlewareError, ModuleInfo, SignatureIndex, Type,
};
use wasmparser::{BlockType, Operator, ValType};

use crate::meter::{STYLUS_ENTRY_POINT, STYLUS_INK_LEFT, STYLUS_INK_STATUS, STYLUS_STACK_LEFT};

const SCRATCH_GLOBAL: &str = "stylus_scratch_global";

fn mw_err(msg: impl Into<String>) -> MiddlewareError {
    MiddlewareError::new("stylus", msg.into())
}

// ── StartMover ──────────────────────────────────────────────────────
//
// Renames the WASM start function to "stylus_start" so it doesn't run at
// module instantiation, then drops all exports except the allowed whitelist.
// Must run before the metering middleware.

const STYLUS_START: &str = "stylus_start";

#[derive(Debug)]
pub struct StartMover {
    debug: bool,
}

impl StartMover {
    pub fn new(debug: bool) -> Self {
        Self { debug }
    }
}

impl ModuleMiddleware for StartMover {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let exports_before = info.exports.len();

        let had_start = if let Some(start) = info.start_function.take() {
            if info.exports.contains_key(STYLUS_START) {
                return Err(mw_err(format!(
                    "function {STYLUS_START} already exists"
                )));
            }
            info.exports
                .insert(STYLUS_START.to_owned(), ExportIndex::Function(start));
            info.function_names.insert(start, STYLUS_START.to_owned());
            true
        } else {
            false
        };

        if had_start && !self.debug {
            return Err(mw_err("start functions not allowed"));
        }

        if !self.debug {
            // Drop all exports except the whitelist (entry point, start, memory).
            info.exports.retain(|name, export| match name.as_str() {
                STYLUS_ENTRY_POINT => matches!(export, ExportIndex::Function(_)),
                STYLUS_START => matches!(export, ExportIndex::Function(_)),
                "memory" => matches!(export, ExportIndex::Memory(_)),
                _ => false,
            });
            info.function_names.clear();
        }
        tracing::warn!(target: "stylus",
            had_start, exports_before, exports_after = info.exports.len(),
            "STARTMOVER applied");
        Ok(())
    }

    fn generate_function_middleware<'a>(&self, _: LocalFunctionIndex) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        Box::new(NoopFunctionMiddleware)
    }
}

#[derive(Debug)]
struct NoopFunctionMiddleware;

impl<'a> FunctionMiddleware<'a> for NoopFunctionMiddleware {
    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        // SAFETY: Operator variants we encounter contain no borrowed data we keep.
        let op_static = unsafe { std::mem::transmute::<Operator<'a>, Operator<'static>>(op) };
        state.push_operator(op_static);
        Ok(())
    }
}

// ── InkMeter ────────────────────────────────────────────────────────

#[derive(Debug)]
pub struct InkMeter {
    header_cost: u64,
    globals: RwLock<Option<[GlobalIndex; 2]>>,
    sigs: RwLock<HashMap<u32, usize>>,
}

impl InkMeter {
    pub fn new(header_cost: u64) -> Self {
        Self {
            header_cost,
            globals: RwLock::new(None),
            sigs: RwLock::new(HashMap::new()),
        }
    }

    fn globals(&self) -> [GlobalIndex; 2] {
        self.globals
            .read()
            .expect("ink globals lock poisoned")
            .expect("missing ink globals")
    }
}

impl ModuleMiddleware for InkMeter {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ink_ty = wasmer_types::GlobalType::new(Type::I64, wasmer_types::Mutability::Var);
        let status_ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);

        let ink_idx = info.globals.push(ink_ty);
        let status_idx = info.globals.push(status_ty);
        info.global_initializers.push(GlobalInit::I64Const(0));
        info.global_initializers.push(GlobalInit::I32Const(0));

        info.exports.insert(
            STYLUS_INK_LEFT.to_string(),
            wasmer_types::ExportIndex::Global(ink_idx),
        );
        info.exports.insert(
            STYLUS_INK_STATUS.to_string(),
            wasmer_types::ExportIndex::Global(status_idx),
        );

        let mut sig_map = self.sigs.write().expect("ink sigs lock poisoned");
        for (sig_idx, sig) in info.signatures.iter() {
            sig_map.insert(sig_idx.as_u32(), sig.params().len());
        }

        *self.globals.write().expect("ink globals lock poisoned") = Some([ink_idx, status_idx]);
        Ok(())
    }

    fn generate_function_middleware<'a>(&self, _: LocalFunctionIndex) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        let [ink, status] = self.globals();
        let sigs = self.sigs.read().expect("ink sigs lock poisoned").clone();
        Box::new(InkMeterFn {
            ink_global: ink,
            status_global: status,
            block: vec![],
            block_cost: 0,
            header_cost: self.header_cost,
            sigs,
        })
    }
}

#[derive(Debug)]
struct InkMeterFn {
    ink_global: GlobalIndex,
    status_global: GlobalIndex,
    block: Vec<Operator<'static>>,
    block_cost: u64,
    header_cost: u64,
    sigs: HashMap<u32, usize>,
}

fn ends_basic_block(op: &Operator) -> bool {
    use Operator::*;
    matches!(
        op,
        End | Else
            | Return
            | Loop { .. }
            | Br { .. }
            | BrTable { .. }
            | BrIf { .. }
            | If { .. }
            | Call { .. }
            | CallIndirect { .. }
    )
}

impl<'a> FunctionMiddleware<'a> for InkMeterFn {
    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        let end = ends_basic_block(&op);
        let op_cost = opcode_ink_cost(&op, &self.sigs);
        let mut cost = self.block_cost.saturating_add(op_cost);
        self.block_cost = cost;

        // SAFETY: Operator variants we support contain no borrowed data.
        // We buffer them as 'static and transmute back when draining.
        let op_static = unsafe { std::mem::transmute::<Operator<'a>, Operator<'static>>(op) };
        self.block.push(op_static);

        if end {
            let ink = self.ink_global.as_u32();
            let status = self.status_global.as_u32();
            cost = cost.saturating_add(self.header_cost);

            state.push_operator(Operator::GlobalGet { global_index: ink });
            state.push_operator(Operator::I64Const { value: cost as i64 });
            state.push_operator(Operator::I64LtU);
            state.push_operator(Operator::If {
                blockty: BlockType::Empty,
            });
            state.push_operator(Operator::I32Const { value: 1 });
            state.push_operator(Operator::GlobalSet {
                global_index: status,
            });
            state.push_operator(Operator::Unreachable);
            state.push_operator(Operator::End);

            state.push_operator(Operator::GlobalGet { global_index: ink });
            state.push_operator(Operator::I64Const { value: cost as i64 });
            state.push_operator(Operator::I64Sub);
            state.push_operator(Operator::GlobalSet { global_index: ink });

            for buffered in self.block.drain(..) {
                let op_a =
                    unsafe { std::mem::transmute::<Operator<'static>, Operator<'a>>(buffered) };
                state.push_operator(op_a);
            }
            self.block_cost = 0;
        }
        Ok(())
    }
}

// ── DynamicMeter ────────────────────────────────────────────────────

#[derive(Debug)]
pub struct DynamicMeter {
    memory_fill_ink: u64,
    memory_copy_ink: u64,
    globals: RwLock<Option<[GlobalIndex; 3]>>,
}

impl DynamicMeter {
    pub fn new(memory_fill_ink: u64, memory_copy_ink: u64) -> Self {
        Self {
            memory_fill_ink,
            memory_copy_ink,
            globals: RwLock::new(None),
        }
    }
}

impl ModuleMiddleware for DynamicMeter {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ink_idx = info
            .exports
            .get(STYLUS_INK_LEFT)
            .and_then(|e| match e {
                wasmer_types::ExportIndex::Global(g) => Some(*g),
                _ => None,
            })
            .ok_or_else(|| mw_err("ink global not found"))?;

        let status_idx = info
            .exports
            .get(STYLUS_INK_STATUS)
            .and_then(|e| match e {
                wasmer_types::ExportIndex::Global(g) => Some(*g),
                _ => None,
            })
            .ok_or_else(|| mw_err("ink status global not found"))?;

        let scratch_ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);
        let scratch_idx = info.globals.push(scratch_ty);
        info.global_initializers.push(GlobalInit::I32Const(0));
        info.exports.insert(
            SCRATCH_GLOBAL.to_string(),
            wasmer_types::ExportIndex::Global(scratch_idx),
        );

        *self.globals.write().expect("dynamic meter lock poisoned") =
            Some([ink_idx, status_idx, scratch_idx]);
        Ok(())
    }

    fn generate_function_middleware<'a>(&self, _: LocalFunctionIndex) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        let globals = self
            .globals
            .read()
            .expect("dynamic meter lock poisoned")
            .expect("missing dynamic globals");
        Box::new(DynamicMeterFn {
            memory_fill_ink: self.memory_fill_ink,
            memory_copy_ink: self.memory_copy_ink,
            globals,
        })
    }
}

#[derive(Debug)]
struct DynamicMeterFn {
    memory_fill_ink: u64,
    memory_copy_ink: u64,
    globals: [GlobalIndex; 3],
}

impl<'a> FunctionMiddleware<'a> for DynamicMeterFn {
    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        use Operator::*;

        let [ink, status, scratch] = self.globals.map(|x| x.as_u32());
        let blockty = BlockType::Empty;

        let coefficient = match &op {
            MemoryFill { .. } => Some(self.memory_fill_ink as i64),
            MemoryCopy { .. } => Some(self.memory_copy_ink as i64),
            _ => None,
        };

        if let Some(coeff) = coefficient {
            // Stack has [dest, val/src, size]. Save size to scratch, compute cost,
            // subtract from ink with overflow check, restore size.
            state.extend([
                GlobalSet {
                    global_index: scratch,
                },
                GlobalGet { global_index: ink },
                GlobalGet { global_index: ink },
                GlobalGet {
                    global_index: scratch,
                },
                I64ExtendI32U,
                I64Const { value: coeff },
                I64Mul,
                I64Sub,
                GlobalSet { global_index: ink },
                GlobalGet { global_index: ink },
                I64LtU,
                If { blockty },
                I32Const { value: 1 },
                GlobalSet {
                    global_index: status,
                },
                Unreachable,
                End,
                GlobalGet {
                    global_index: scratch,
                },
            ]);
        }

        state.push_operator(op);
        Ok(())
    }
}

// ── DepthChecker ────────────────────────────────────────────────────

type FuncMap = HashMap<FunctionIndex, FunctionType>;
type SigMap = HashMap<SignatureIndex, FunctionType>;

#[derive(Debug)]
pub struct DepthChecker {
    frame_limit: u32,
    frame_contention: u16,
    global: RwLock<Option<GlobalIndex>>,
    funcs: RwLock<Option<FuncMap>>,
    sigs: RwLock<Option<SigMap>>,
}

impl DepthChecker {
    pub fn new(frame_limit: u32, frame_contention: u16) -> Self {
        Self {
            frame_limit,
            frame_contention,
            global: RwLock::new(None),
            funcs: RwLock::new(None),
            sigs: RwLock::new(None),
        }
    }
}

impl ModuleMiddleware for DepthChecker {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);
        let idx = info.globals.push(ty);
        info.global_initializers.push(GlobalInit::I32Const(0));
        info.exports.insert(
            STYLUS_STACK_LEFT.to_string(),
            wasmer_types::ExportIndex::Global(idx),
        );

        let mut funcs = HashMap::new();
        for (func_idx, sig_idx) in info.functions.iter() {
            if let Some(sig) = info.signatures.get(*sig_idx) {
                funcs.insert(func_idx, sig.clone());
            }
        }
        let mut sigs = HashMap::new();
        for (sig_idx, sig) in info.signatures.iter() {
            sigs.insert(sig_idx, sig.clone());
        }

        *self.global.write().expect("depth checker lock poisoned") = Some(idx);
        *self.funcs.write().expect("depth checker lock poisoned") = Some(funcs);
        *self.sigs.write().expect("depth checker lock poisoned") = Some(sigs);
        Ok(())
    }

    fn generate_function_middleware<'a>(
        &self,
        _: LocalFunctionIndex,
    ) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        let g = self
            .global
            .read()
            .expect("depth checker lock poisoned")
            .expect("missing depth global");
        let funcs = self
            .funcs
            .read()
            .expect("depth checker lock poisoned")
            .clone()
            .expect("missing funcs");
        let sigs = self
            .sigs
            .read()
            .expect("depth checker lock poisoned")
            .clone()
            .expect("missing sigs");
        Box::new(DepthCheckerFn {
            global: g,
            funcs,
            sigs,
            locals: None,
            frame_limit: self.frame_limit,
            frame_contention: self.frame_contention,
            scopes: 1,
            code: vec![],
            done: false,
        })
    }
}

#[derive(Debug)]
struct DepthCheckerFn {
    global: GlobalIndex,
    funcs: FuncMap,
    sigs: SigMap,
    locals: Option<usize>,
    frame_limit: u32,
    frame_contention: u16,
    scopes: isize,
    code: Vec<Operator<'static>>,
    done: bool,
}

impl DepthCheckerFn {
    #[rustfmt::skip]
    fn worst_case_depth(&self) -> Result<u32, MiddlewareError> {
        use Operator::*;

        let mut worst: u32 = 0;
        let mut stack: u32 = 0;

        macro_rules! push {
            ($count:expr) => {{ stack += $count; worst = worst.max(stack); }};
            () => { push!(1) };
        }
        macro_rules! pop {
            ($count:expr) => {{ stack = stack.saturating_sub($count); }};
            () => { pop!(1) };
        }
        macro_rules! ins_and_outs {
            ($ty:expr) => {{
                let ins = $ty.params().len() as u32;
                let outs = $ty.results().len() as u32;
                push!(outs);
                pop!(ins);
            }};
        }
        macro_rules! op {
            ($first:ident $(,$opcode:ident)* $(,)?) => {
                $first $(| $opcode)*
            };
        }
        macro_rules! dot {
            ($first:ident $(,$opcode:ident)* $(,)?) => {
                $first { .. } $(| $opcode { .. })*
            };
        }
        macro_rules! block_type {
            ($ty:expr) => {{
                match $ty {
                    BlockType::Empty => {}
                    BlockType::Type(_) => push!(1),
                    BlockType::FuncType(id) => {
                        let index = SignatureIndex::from_u32(*id);
                        let Some(ty) = self.sigs.get(&index) else {
                            return Err(mw_err(format!("missing type for func {id}")));
                        };
                        ins_and_outs!(ty);
                    }
                }
            }};
        }

        let mut scopes = vec![stack];

        for op in &self.code {
            match op {
                Block { blockty } => {
                    block_type!(blockty);
                    scopes.push(stack);
                }
                Loop { blockty } => {
                    block_type!(blockty);
                    scopes.push(stack);
                }
                If { blockty } => {
                    pop!();
                    block_type!(blockty);
                    scopes.push(stack);
                }
                Else => {
                    stack = match scopes.last() {
                        Some(scope) => *scope,
                        None => return Err(mw_err("malformed if-else scope")),
                    };
                }
                End => {
                    stack = match scopes.pop() {
                        Some(stack) => stack,
                        None => return Err(mw_err("malformed scoping at end of block")),
                    };
                }

                Call { function_index } => {
                    let index = FunctionIndex::from_u32(*function_index);
                    let Some(ty) = self.funcs.get(&index) else {
                        return Err(mw_err(format!("missing type for func {function_index}")));
                    };
                    ins_and_outs!(ty)
                }
                CallIndirect { type_index, .. } => {
                    let index = SignatureIndex::from_u32(*type_index);
                    let Some(ty) = self.sigs.get(&index) else {
                        return Err(mw_err(format!("missing type for signature {type_index}")));
                    };
                    ins_and_outs!(ty);
                    pop!() // table index
                }

                MemoryFill { .. } | MemoryCopy { .. } => pop!(3), // 3 args, 0 returns

                op!(
                    Nop, Unreachable,
                    I32Eqz, I64Eqz, I32Clz, I32Ctz, I32Popcnt, I64Clz, I64Ctz, I64Popcnt,
                )
                | dot!(
                    Br, Return,
                    LocalTee, MemoryGrow,
                    I32Load, I64Load, F32Load, F64Load,
                    I32Load8S, I32Load8U, I32Load16S, I32Load16U, I64Load8S, I64Load8U,
                    I64Load16S, I64Load16U, I64Load32S, I64Load32U,
                    I32WrapI64, I64ExtendI32S, I64ExtendI32U,
                    I32Extend8S, I32Extend16S, I64Extend8S, I64Extend16S, I64Extend32S,
                    F32Abs, F32Neg, F32Ceil, F32Floor, F32Trunc, F32Nearest, F32Sqrt,
                    F64Abs, F64Neg, F64Ceil, F64Floor, F64Trunc, F64Nearest, F64Sqrt,
                    I32TruncF32S, I32TruncF32U, I32TruncF64S, I32TruncF64U,
                    I64TruncF32S, I64TruncF32U, I64TruncF64S, I64TruncF64U,
                    F32ConvertI32S, F32ConvertI32U, F32ConvertI64S, F32ConvertI64U, F32DemoteF64,
                    F64ConvertI32S, F64ConvertI32U, F64ConvertI64S, F64ConvertI64U, F64PromoteF32,
                    I32ReinterpretF32, I64ReinterpretF64, F32ReinterpretI32, F64ReinterpretI64,
                    I32TruncSatF32S, I32TruncSatF32U, I32TruncSatF64S, I32TruncSatF64U,
                    I64TruncSatF32S, I64TruncSatF32U, I64TruncSatF64S, I64TruncSatF64U,
                ) => {}

                dot!(
                    LocalGet, GlobalGet, MemorySize,
                    I32Const, I64Const, F32Const, F64Const,
                ) => push!(),

                op!(
                    Drop,
                    I32Eq, I32Ne, I32LtS, I32LtU, I32GtS, I32GtU, I32LeS, I32LeU, I32GeS, I32GeU,
                    I64Eq, I64Ne, I64LtS, I64LtU, I64GtS, I64GtU, I64LeS, I64LeU, I64GeS, I64GeU,
                    F32Eq, F32Ne, F32Lt, F32Gt, F32Le, F32Ge,
                    F64Eq, F64Ne, F64Lt, F64Gt, F64Le, F64Ge,
                    I32Add, I32Sub, I32Mul, I32DivS, I32DivU, I32RemS, I32RemU,
                    I64Add, I64Sub, I64Mul, I64DivS, I64DivU, I64RemS, I64RemU,
                    I32And, I32Or, I32Xor, I32Shl, I32ShrS, I32ShrU, I32Rotl, I32Rotr,
                    I64And, I64Or, I64Xor, I64Shl, I64ShrS, I64ShrU, I64Rotl, I64Rotr,
                    F32Add, F32Sub, F32Mul, F32Div, F32Min, F32Max, F32Copysign,
                    F64Add, F64Sub, F64Mul, F64Div, F64Min, F64Max, F64Copysign,
                )
                | dot!(BrIf, BrTable, LocalSet, GlobalSet) => pop!(),

                dot!(
                    Select,
                    I32Store, I64Store, F32Store, F64Store,
                    I32Store8, I32Store16, I64Store8, I64Store16, I64Store32,
                ) => pop!(2),

                unsupported @ dot!(Try, Catch, Throw, Rethrow, ThrowRef, TryTable) => {
                    return Err(mw_err(format!("exception-handling not supported {unsupported:?}")));
                }
                unsupported @ dot!(ReturnCall, ReturnCallIndirect) => {
                    return Err(mw_err(format!("tail-call not supported {unsupported:?}")));
                }
                unsupported @ dot!(CallRef, ReturnCallRef) => {
                    return Err(mw_err(format!("typed function references not supported {unsupported:?}")));
                }
                unsupported @ (dot!(Delegate) | op!(CatchAll)) => {
                    return Err(mw_err(format!("exception-handling not supported {unsupported:?}")));
                }
                unsupported @ (op!(RefIsNull) | dot!(TypedSelect, RefNull, RefFunc, RefEq)) => {
                    return Err(mw_err(format!("reference-types not supported {unsupported:?}")));
                }
                unsupported @ dot!(RefAsNonNull, BrOnNull, BrOnNonNull) => {
                    return Err(mw_err(format!("typed function references not supported {unsupported:?}")));
                }
                unsupported @ dot!(
                    MemoryInit, DataDrop, TableInit, ElemDrop,
                    TableCopy, TableFill, TableGet, TableSet, TableGrow, TableSize
                ) => {
                    return Err(mw_err(format!("bulk-memory not fully supported {unsupported:?}")));
                }
                unsupported @ dot!(MemoryDiscard) => {
                    return Err(mw_err(format!("memory discard not supported {unsupported:?}")));
                }
                unsupported @ dot!(
                    StructNew, StructNewDefault, StructGet, StructGetS, StructGetU, StructSet,
                    ArrayNew, ArrayNewDefault, ArrayNewFixed, ArrayNewData, ArrayNewElem,
                    ArrayGet, ArrayGetS, ArrayGetU, ArraySet, ArrayLen, ArrayFill, ArrayCopy,
                    ArrayInitData, ArrayInitElem,
                    RefTestNonNull, RefTestNullable, RefCastNonNull, RefCastNullable,
                    BrOnCast, BrOnCastFail, AnyConvertExtern, ExternConvertAny,
                    RefI31, I31GetS, I31GetU
                ) => {
                    return Err(mw_err(format!("GC extension not supported {unsupported:?}")));
                }
                unsupported @ dot!(
                    MemoryAtomicNotify, MemoryAtomicWait32, MemoryAtomicWait64, AtomicFence,
                    I32AtomicLoad, I64AtomicLoad, I32AtomicLoad8U, I32AtomicLoad16U,
                    I64AtomicLoad8U, I64AtomicLoad16U, I64AtomicLoad32U,
                    I32AtomicStore, I64AtomicStore, I32AtomicStore8, I32AtomicStore16,
                    I64AtomicStore8, I64AtomicStore16, I64AtomicStore32,
                    I32AtomicRmwAdd, I64AtomicRmwAdd, I32AtomicRmw8AddU, I32AtomicRmw16AddU,
                    I64AtomicRmw8AddU, I64AtomicRmw16AddU, I64AtomicRmw32AddU,
                    I32AtomicRmwSub, I64AtomicRmwSub, I32AtomicRmw8SubU, I32AtomicRmw16SubU,
                    I64AtomicRmw8SubU, I64AtomicRmw16SubU, I64AtomicRmw32SubU,
                    I32AtomicRmwAnd, I64AtomicRmwAnd, I32AtomicRmw8AndU, I32AtomicRmw16AndU,
                    I64AtomicRmw8AndU, I64AtomicRmw16AndU, I64AtomicRmw32AndU,
                    I32AtomicRmwOr, I64AtomicRmwOr, I32AtomicRmw8OrU, I32AtomicRmw16OrU,
                    I64AtomicRmw8OrU, I64AtomicRmw16OrU, I64AtomicRmw32OrU,
                    I32AtomicRmwXor, I64AtomicRmwXor, I32AtomicRmw8XorU, I32AtomicRmw16XorU,
                    I64AtomicRmw8XorU, I64AtomicRmw16XorU, I64AtomicRmw32XorU,
                    I32AtomicRmwXchg, I64AtomicRmwXchg, I32AtomicRmw8XchgU, I32AtomicRmw16XchgU,
                    I64AtomicRmw8XchgU, I64AtomicRmw16XchgU, I64AtomicRmw32XchgU,
                    I32AtomicRmwCmpxchg, I64AtomicRmwCmpxchg, I32AtomicRmw8CmpxchgU,
                    I32AtomicRmw16CmpxchgU, I64AtomicRmw8CmpxchgU, I64AtomicRmw16CmpxchgU,
                    I64AtomicRmw32CmpxchgU
                ) => {
                    return Err(mw_err(format!("threads extension not supported {unsupported:?}")));
                }
                unsupported @ dot!(
                    V128Load, V128Load8x8S, V128Load8x8U, V128Load16x4S, V128Load16x4U,
                    V128Load32x2S, V128Load8Splat, V128Load16Splat, V128Load32Splat,
                    V128Load64Splat, V128Load32Zero, V128Load64Zero, V128Load32x2U,
                    V128Store, V128Load8Lane, V128Load16Lane, V128Load32Lane, V128Load64Lane,
                    V128Store8Lane, V128Store16Lane, V128Store32Lane, V128Store64Lane, V128Const,
                    I8x16Shuffle, I8x16ExtractLaneS, I8x16ExtractLaneU, I8x16ReplaceLane,
                    I16x8ExtractLaneS, I16x8ExtractLaneU, I16x8ReplaceLane,
                    I32x4ExtractLane, I32x4ReplaceLane, I64x2ExtractLane, I64x2ReplaceLane,
                    F32x4ExtractLane, F32x4ReplaceLane, F64x2ExtractLane, F64x2ReplaceLane,
                    I8x16Swizzle, I8x16Splat, I16x8Splat, I32x4Splat, I64x2Splat,
                    F32x4Splat, F64x2Splat,
                    I8x16Eq, I8x16Ne, I8x16LtS, I8x16LtU, I8x16GtS, I8x16GtU,
                    I8x16LeS, I8x16LeU, I8x16GeS, I8x16GeU,
                    I16x8Eq, I16x8Ne, I16x8LtS, I16x8LtU, I16x8GtS, I16x8GtU,
                    I16x8LeS, I16x8LeU, I16x8GeS, I16x8GeU,
                    I32x4Eq, I32x4Ne, I32x4LtS, I32x4LtU, I32x4GtS, I32x4GtU,
                    I32x4LeS, I32x4LeU, I32x4GeS, I32x4GeU,
                    I64x2Eq, I64x2Ne, I64x2LtS, I64x2GtS, I64x2LeS, I64x2GeS,
                    F32x4Eq, F32x4Ne, F32x4Lt, F32x4Gt, F32x4Le, F32x4Ge,
                    F64x2Eq, F64x2Ne, F64x2Lt, F64x2Gt, F64x2Le, F64x2Ge,
                    V128Not, V128And, V128AndNot, V128Or, V128Xor, V128Bitselect, V128AnyTrue,
                    I8x16Abs, I8x16Neg, I8x16Popcnt, I8x16AllTrue, I8x16Bitmask,
                    I8x16NarrowI16x8S, I8x16NarrowI16x8U,
                    I8x16Shl, I8x16ShrS, I8x16ShrU, I8x16Add, I8x16AddSatS, I8x16AddSatU,
                    I8x16Sub, I8x16SubSatS, I8x16SubSatU, I8x16MinS, I8x16MinU,
                    I8x16MaxS, I8x16MaxU, I8x16AvgrU,
                    I16x8ExtAddPairwiseI8x16S, I16x8ExtAddPairwiseI8x16U, I16x8Abs, I16x8Neg,
                    I16x8Q15MulrSatS, I16x8AllTrue, I16x8Bitmask,
                    I16x8NarrowI32x4S, I16x8NarrowI32x4U,
                    I16x8ExtendLowI8x16S, I16x8ExtendHighI8x16S,
                    I16x8ExtendLowI8x16U, I16x8ExtendHighI8x16U,
                    I16x8Shl, I16x8ShrS, I16x8ShrU, I16x8Add, I16x8AddSatS, I16x8AddSatU,
                    I16x8Sub, I16x8SubSatS, I16x8SubSatU, I16x8Mul,
                    I16x8MinS, I16x8MinU, I16x8MaxS, I16x8MaxU, I16x8AvgrU,
                    I16x8ExtMulLowI8x16S, I16x8ExtMulHighI8x16S,
                    I16x8ExtMulLowI8x16U, I16x8ExtMulHighI8x16U,
                    I32x4ExtAddPairwiseI16x8U, I32x4Abs, I32x4Neg, I32x4AllTrue, I32x4Bitmask,
                    I32x4ExtAddPairwiseI16x8S,
                    I32x4ExtendLowI16x8S, I32x4ExtendHighI16x8S,
                    I32x4ExtendLowI16x8U, I32x4ExtendHighI16x8U,
                    I32x4Shl, I32x4ShrS, I32x4ShrU, I32x4Add, I32x4Sub, I32x4Mul,
                    I32x4MinS, I32x4MinU, I32x4MaxS, I32x4MaxU, I32x4DotI16x8S,
                    I32x4ExtMulLowI16x8S, I32x4ExtMulHighI16x8S,
                    I32x4ExtMulLowI16x8U, I32x4ExtMulHighI16x8U,
                    I64x2Abs, I64x2Neg, I64x2AllTrue, I64x2Bitmask,
                    I64x2ExtendLowI32x4S, I64x2ExtendHighI32x4S,
                    I64x2ExtendLowI32x4U, I64x2ExtendHighI32x4U,
                    I64x2Shl, I64x2ShrS, I64x2ShrU, I64x2Add, I64x2Sub, I64x2Mul,
                    I64x2ExtMulLowI32x4S, I64x2ExtMulHighI32x4S,
                    I64x2ExtMulLowI32x4U, I64x2ExtMulHighI32x4U,
                    F32x4Ceil, F32x4Floor, F32x4Trunc, F32x4Nearest,
                    F32x4Abs, F32x4Neg, F32x4Sqrt, F32x4Add, F32x4Sub, F32x4Mul, F32x4Div,
                    F32x4Min, F32x4Max, F32x4PMin, F32x4PMax,
                    F64x2Ceil, F64x2Floor, F64x2Trunc, F64x2Nearest,
                    F64x2Abs, F64x2Neg, F64x2Sqrt, F64x2Add, F64x2Sub, F64x2Mul, F64x2Div,
                    F64x2Min, F64x2Max, F64x2PMin, F64x2PMax,
                    I32x4TruncSatF32x4S, I32x4TruncSatF32x4U,
                    F32x4ConvertI32x4S, F32x4ConvertI32x4U,
                    I32x4TruncSatF64x2SZero, I32x4TruncSatF64x2UZero,
                    F64x2ConvertLowI32x4S, F64x2ConvertLowI32x4U,
                    F32x4DemoteF64x2Zero, F64x2PromoteLowF32x4,
                    I8x16RelaxedSwizzle,
                    I32x4RelaxedTruncF32x4S, I32x4RelaxedTruncF32x4U,
                    I32x4RelaxedTruncF64x2SZero, I32x4RelaxedTruncF64x2UZero,
                    F32x4RelaxedMadd, F32x4RelaxedNmadd, F64x2RelaxedMadd, F64x2RelaxedNmadd,
                    I8x16RelaxedLaneselect, I16x8RelaxedLaneselect,
                    I32x4RelaxedLaneselect, I64x2RelaxedLaneselect,
                    F32x4RelaxedMin, F32x4RelaxedMax, F64x2RelaxedMin, F64x2RelaxedMax,
                    I16x8RelaxedQ15mulrS, I16x8RelaxedDotI8x16I7x16S,
                    I32x4RelaxedDotI8x16I7x16AddS
                ) => {
                    return Err(mw_err(format!("SIMD extension not supported {unsupported:?}")));
                }
            };
        }

        if self.locals.is_none() {
            return Err(mw_err("missing locals info"));
        }

        let contention = worst;
        if contention > self.frame_contention.into() {
            return Err(mw_err(format!(
                "too many values on the stack at once: {contention} > {}",
                self.frame_contention
            )));
        }

        let locals = self.locals.unwrap_or_default();
        Ok(worst + locals as u32 + 4)
    }
}

impl<'a> FunctionMiddleware<'a> for DepthCheckerFn {
    fn locals_info(&mut self, locals: &[ValType]) {
        self.locals = Some(locals.len());
    }

    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        if self.done {
            return Err(mw_err("depth checker: feed called after finalization"));
        }

        match op {
            Operator::Block { .. } | Operator::Loop { .. } | Operator::If { .. } => {
                self.scopes += 1;
            }
            Operator::End => {
                self.scopes -= 1;
            }
            _ => {}
        }
        if self.scopes < 0 {
            return Err(mw_err("malformed scoping detected"));
        }

        let last = self.scopes == 0 && matches!(op, Operator::End);

        // SAFETY: Operator variants we support contain no borrowed data.
        let op_static =
            unsafe { std::mem::transmute::<Operator<'a>, Operator<'static>>(op) };
        self.code.push(op_static);

        if !last {
            return Ok(());
        }

        let size = self.worst_case_depth()?;
        let g = self.global.as_u32();

        if size > self.frame_limit {
            return Err(mw_err(format!(
                "frame too large: {size} > {}-word limit",
                self.frame_limit
            )));
        }

        // Prologue: check and deduct depth budget
        state.extend([
            Operator::GlobalGet { global_index: g },
            Operator::I32Const {
                value: size as i32,
            },
            Operator::I32LeU,
            Operator::If {
                blockty: BlockType::Empty,
            },
            Operator::I32Const { value: 0 },
            Operator::GlobalSet { global_index: g },
            Operator::Unreachable,
            Operator::End,
            Operator::GlobalGet { global_index: g },
            Operator::I32Const {
                value: size as i32,
            },
            Operator::I32Sub,
            Operator::GlobalSet { global_index: g },
        ]);

        // Insert an extraneous Return before the final End to match Arbitrator.
        let mut code = std::mem::take(&mut self.code);
        let final_end = code.pop().unwrap();
        code.push(Operator::Return);
        code.push(final_end);

        for op_s in code {
            let is_return = matches!(op_s, Operator::Return);
            if is_return {
                state.extend([
                    Operator::GlobalGet { global_index: g },
                    Operator::I32Const {
                        value: size as i32,
                    },
                    Operator::I32Add,
                    Operator::GlobalSet { global_index: g },
                ]);
            }
            let op_a =
                unsafe { std::mem::transmute::<Operator<'static>, Operator<'a>>(op_s) };
            state.push_operator(op_a);
        }

        self.done = true;
        Ok(())
    }
}

// ── HeapBound ───────────────────────────────────────────────────────

#[derive(Debug)]
pub struct HeapBound {
    globals: RwLock<Option<(GlobalIndex, Option<FunctionIndex>)>>,
}

impl HeapBound {
    pub fn new() -> Self {
        Self {
            globals: RwLock::new(None),
        }
    }
}

impl ModuleMiddleware for HeapBound {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let scratch_idx = info
            .exports
            .get(SCRATCH_GLOBAL)
            .and_then(|e| match e {
                wasmer_types::ExportIndex::Global(g) => Some(*g),
                _ => None,
            })
            .ok_or_else(|| mw_err("scratch global not found"))?;

        let pay_func = info.imports.iter().find_map(|(key, idx)| {
            if key.field == "pay_for_memory_grow" {
                if let ImportIndex::Function(f) = idx {
                    return Some(*f);
                }
            }
            None
        });

        *self.globals.write().expect("heap bound lock poisoned") = Some((scratch_idx, pay_func));
        Ok(())
    }

    fn generate_function_middleware<'a>(&self, _: LocalFunctionIndex) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        let (scratch, pay_func) = self
            .globals
            .read()
            .expect("heap bound lock poisoned")
            .expect("missing heap globals");
        Box::new(HeapBoundFn { scratch, pay_func })
    }
}

#[derive(Debug)]
struct HeapBoundFn {
    scratch: GlobalIndex,
    pay_func: Option<FunctionIndex>,
}

impl<'a> FunctionMiddleware<'a> for HeapBoundFn {
    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        if let (Operator::MemoryGrow { .. }, Some(pay)) = (&op, self.pay_func) {
            let g = self.scratch.as_u32();
            let f = pay.as_u32();
            state.extend([
                Operator::GlobalSet { global_index: g },
                Operator::GlobalGet { global_index: g },
                Operator::GlobalGet { global_index: g },
                Operator::Call { function_index: f },
            ]);
        }
        state.push_operator(op);
        Ok(())
    }
}

// ── Opcode ink costs ────────────────────────────────────────────────

/// Per-opcode ink cost used by the ink meter middleware.
#[rustfmt::skip]
pub fn opcode_ink_cost(op: &Operator, sigs: &HashMap<u32, usize>) -> u64 {
    use Operator::*;

    macro_rules! op {
        ($first:ident $(,$opcode:ident)*) => { $first $(| $opcode)* };
    }
    macro_rules! dot {
        ($first:ident $(,$opcode:ident)*) => { $first { .. } $(| $opcode { .. })* };
    }

    match op {
        op!(Unreachable, Return) => 1,
        op!(Nop) | dot!(I32Const, I64Const) => 1,
        op!(Drop) => 9,

        dot!(Block, Loop) | op!(Else, End) => 1,
        dot!(Br, BrIf, If) => 765,
        dot!(Select) => 1250,
        dot!(Call) => 3800,
        dot!(LocalGet, LocalTee) => 75,
        dot!(LocalSet) => 210,
        dot!(GlobalGet) => 225,
        dot!(GlobalSet) => 575,
        dot!(I32Load, I32Load8S, I32Load8U, I32Load16S, I32Load16U) => 670,
        dot!(I64Load, I64Load8S, I64Load8U, I64Load16S, I64Load16U, I64Load32S, I64Load32U) => 680,
        dot!(I32Store, I32Store8, I32Store16) => 825,
        dot!(I64Store, I64Store8, I64Store16, I64Store32) => 950,
        dot!(MemorySize) => 3000,
        dot!(MemoryGrow) => 8050,

        op!(I32Eqz, I32Eq, I32Ne, I32LtS, I32LtU, I32GtS, I32GtU, I32LeS, I32LeU, I32GeS, I32GeU) => 170,
        op!(I64Eqz, I64Eq, I64Ne, I64LtS, I64LtU, I64GtS, I64GtU, I64LeS, I64LeU, I64GeS, I64GeU) => 225,

        op!(I32Clz, I32Ctz) => 210,
        op!(I32Add, I32Sub) => 70,
        op!(I32Mul) => 160,
        op!(I32DivS, I32DivU, I32RemS, I32RemU) => 1120,
        op!(I32And, I32Or, I32Xor, I32Shl, I32ShrS, I32ShrU, I32Rotl, I32Rotr) => 70,

        op!(I64Clz, I64Ctz) => 210,
        op!(I64Add, I64Sub) => 100,
        op!(I64Mul) => 160,
        op!(I64DivS, I64DivU, I64RemS, I64RemU) => 1270,
        op!(I64And, I64Or, I64Xor, I64Shl, I64ShrS, I64ShrU, I64Rotl, I64Rotr) => 100,

        op!(I32Popcnt) => 2650,
        op!(I64Popcnt) => 6000,

        op!(I32WrapI64, I64ExtendI32S, I64ExtendI32U) => 100,
        op!(I32Extend8S, I32Extend16S, I64Extend8S, I64Extend16S, I64Extend32S) => 100,
        dot!(MemoryCopy) => 950,
        dot!(MemoryFill) => 950,

        BrTable { targets } => 2400 + 325 * targets.len() as u64,
        CallIndirect { type_index, .. } => {
            let params = sigs.get(type_index).copied().unwrap_or(0);
            13610 + 650 * params as u64
        },

        _ => u64::MAX,
    }
}

