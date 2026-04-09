use std::{collections::HashMap, sync::RwLock};

use wasmer_compiler::{FunctionMiddleware, MiddlewareReaderState, ModuleMiddleware};
use wasmer_types::{
    ExportIndex, FunctionIndex, GlobalIndex, GlobalInit, ImportIndex, LocalFunctionIndex,
    MiddlewareError, ModuleInfo, Type,
};
use wasmparser::{BlockType, Operator};

use crate::meter::{STYLUS_ENTRY_POINT, STYLUS_INK_LEFT, STYLUS_INK_STATUS, STYLUS_STACK_LEFT};

const SCRATCH_GLOBAL: &str = "stylus_scratch_global";

fn mw_err(msg: impl Into<String>) -> MiddlewareError {
    MiddlewareError::new("stylus", msg.into())
}

// ── StartMover ──────────────────────────────────────────────────────
//
// Renames the WASM start function to "stylus_start" so it doesn't run
// automatically at module instantiation, then drops all exports except the
// allowed whitelist. Mirrors Nitro's `start::StartMover`. Must run BEFORE
// the metering middleware so the meter sees the modified module.

/// Name the start function is renamed to (matches Nitro).
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

        // Move the start function to a named export, mirroring Nitro.
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

#[derive(Debug)]
pub struct DepthChecker {
    max_depth: u32,
    global: RwLock<Option<GlobalIndex>>,
}

impl DepthChecker {
    pub fn new(max_depth: u32) -> Self {
        Self {
            max_depth,
            global: RwLock::new(None),
        }
    }
}

impl ModuleMiddleware for DepthChecker {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);
        let idx = info.globals.push(ty);
        info.global_initializers
            .push(GlobalInit::I32Const(self.max_depth as i32));
        info.exports.insert(
            STYLUS_STACK_LEFT.to_string(),
            wasmer_types::ExportIndex::Global(idx),
        );
        *self.global.write().expect("depth checker lock poisoned") = Some(idx);
        Ok(())
    }

    fn generate_function_middleware<'a>(&self, _: LocalFunctionIndex) -> Box<dyn FunctionMiddleware<'a> + 'a> {
        let g = self
            .global
            .read()
            .expect("depth checker lock poisoned")
            .expect("missing depth global");
        Box::new(DepthCheckerFn {
            global: g,
            frame_cost: 1,
            emitted_entry: false,
        })
    }
}

#[derive(Debug)]
struct DepthCheckerFn {
    global: GlobalIndex,
    frame_cost: u32,
    emitted_entry: bool,
}

impl<'a> FunctionMiddleware<'a> for DepthCheckerFn {
    fn feed(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        if !self.emitted_entry {
            self.emitted_entry = true;
            let g = self.global.as_u32();
            let cost = self.frame_cost as i32;

            state.extend([
                Operator::GlobalGet { global_index: g },
                Operator::I32Const { value: cost },
                Operator::I32LeU,
                Operator::If {
                    blockty: BlockType::Empty,
                },
                Operator::Unreachable,
                Operator::End,
                Operator::GlobalGet { global_index: g },
                Operator::I32Const { value: cost },
                Operator::I32Sub,
                Operator::GlobalSet { global_index: g },
            ]);
        }

        if matches!(op, Operator::Return) {
            let g = self.global.as_u32();
            state.extend([
                Operator::GlobalGet { global_index: g },
                Operator::I32Const {
                    value: self.frame_cost as i32,
                },
                Operator::I32Add,
                Operator::GlobalSet { global_index: g },
            ]);
        }

        state.push_operator(op);
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

// ── Opcode ink costs (matches Nitro pricing_v1) ─────────────────────

#[rustfmt::skip]
fn opcode_ink_cost(op: &Operator, sigs: &HashMap<u32, usize>) -> u64 {
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

