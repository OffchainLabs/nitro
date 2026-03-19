use std::sync::RwLock;

use wasmer_compiler::{FunctionMiddleware, MiddlewareReaderState, ModuleMiddleware};
use wasmer_types::{
    GlobalIndex, GlobalInit, LocalFunctionIndex, MiddlewareError, ModuleInfo, Type,
};
use wasmparser::{BlockType, Operator};

use crate::meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS, STYLUS_STACK_LEFT};

// ── InkMeter ────────────────────────────────────────────────────────

#[derive(Debug)]
pub struct InkMeter {
    header_cost: u64,
    globals: RwLock<Option<[GlobalIndex; 2]>>,
}

impl InkMeter {
    pub fn new(header_cost: u64) -> Self {
        Self {
            header_cost,
            globals: RwLock::new(None),
        }
    }

    fn globals(&self) -> [GlobalIndex; 2] {
        self.globals.read().unwrap().expect("missing ink globals")
    }
}

impl ModuleMiddleware for InkMeter {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ink_ty = wasmer_types::GlobalType::new(Type::I64, wasmer_types::Mutability::Var);
        let status_ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);

        let ink_idx = info.globals.push(ink_ty);
        let status_idx = info.globals.push(status_ty);

        info.global_initializers
            .push(GlobalInit::I64Const(0))
            .as_u32();
        info.global_initializers
            .push(GlobalInit::I32Const(0))
            .as_u32();

        info.exports
            .insert(STYLUS_INK_LEFT.to_string(), wasmer_types::ExportIndex::Global(ink_idx));
        info.exports
            .insert(STYLUS_INK_STATUS.to_string(), wasmer_types::ExportIndex::Global(status_idx));

        *self.globals.write().unwrap() = Some([ink_idx, status_idx]);
        Ok(())
    }

    fn generate_function_middleware(
        &self,
        _: LocalFunctionIndex,
    ) -> Box<dyn FunctionMiddleware> {
        let [ink, status] = self.globals();
        Box::new(InkMeterFn {
            ink_global: ink,
            status_global: status,
            block: vec![],
            block_cost: 0,
            header_cost: self.header_cost,
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

impl FunctionMiddleware for InkMeterFn {
    fn feed<'a>(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        let end = ends_basic_block(&op);

        let op_cost = opcode_ink_cost(&op);
        let mut cost = self.block_cost.saturating_add(op_cost);
        self.block_cost = cost;

        // Safety: we only hold these operators until the end of the basic block,
        // then drain them into `state` which has lifetime 'a. The operators are
        // re-created as 'static using the same variant constructors (no borrowed data
        // in the variants we support).
        let op_static = unsafe { std::mem::transmute::<Operator<'a>, Operator<'static>>(op) };
        self.block.push(op_static);

        if end {
            let ink = self.ink_global.as_u32();
            let status = self.status_global.as_u32();

            cost = cost.saturating_add(self.header_cost);

            state.push_operator(Operator::GlobalGet { global_index: ink });
            state.push_operator(Operator::I64Const {
                value: cost as i64,
            });
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
            state.push_operator(Operator::I64Const {
                value: cost as i64,
            });
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

// ── DepthChecker ────────────────────────────────────────────────────

#[derive(Debug)]
pub struct DepthChecker {
    max_depth: u32,
    globals: RwLock<Option<GlobalIndex>>,
}

impl DepthChecker {
    pub fn new(max_depth: u32) -> Self {
        Self {
            max_depth,
            globals: RwLock::new(None),
        }
    }

    fn global(&self) -> GlobalIndex {
        self.globals.read().unwrap().expect("missing depth global")
    }
}

impl ModuleMiddleware for DepthChecker {
    fn transform_module_info(&self, info: &mut ModuleInfo) -> Result<(), MiddlewareError> {
        let ty = wasmer_types::GlobalType::new(Type::I32, wasmer_types::Mutability::Var);
        let idx = info.globals.push(ty);
        info.global_initializers
            .push(GlobalInit::I32Const(self.max_depth as i32));
        info.exports
            .insert(STYLUS_STACK_LEFT.to_string(), wasmer_types::ExportIndex::Global(idx));
        *self.globals.write().unwrap() = Some(idx);
        Ok(())
    }

    fn generate_function_middleware(
        &self,
        _: LocalFunctionIndex,
    ) -> Box<dyn FunctionMiddleware> {
        Box::new(DepthCheckerFn {
            global: self.global(),
            // Conservative estimate: each function uses 1 depth unit.
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

impl FunctionMiddleware for DepthCheckerFn {
    fn feed<'a>(
        &mut self,
        op: Operator<'a>,
        state: &mut MiddlewareReaderState<'a>,
    ) -> Result<(), MiddlewareError> {
        if !self.emitted_entry {
            self.emitted_entry = true;
            let g = self.global.as_u32();
            let cost = self.frame_cost;

            // if stack_left < cost: trap
            state.push_operator(Operator::GlobalGet { global_index: g });
            state.push_operator(Operator::I32Const {
                value: cost as i32,
            });
            state.push_operator(Operator::I32LeU);
            state.push_operator(Operator::If {
                blockty: BlockType::Empty,
            });
            state.push_operator(Operator::Unreachable);
            state.push_operator(Operator::End);

            // stack_left -= cost
            state.push_operator(Operator::GlobalGet { global_index: g });
            state.push_operator(Operator::I32Const {
                value: cost as i32,
            });
            state.push_operator(Operator::I32Sub);
            state.push_operator(Operator::GlobalSet { global_index: g });
        }

        // On return, re-credit the frame cost.
        if matches!(op, Operator::Return) {
            let g = self.global.as_u32();
            let cost = self.frame_cost;
            state.push_operator(Operator::GlobalGet { global_index: g });
            state.push_operator(Operator::I32Const {
                value: cost as i32,
            });
            state.push_operator(Operator::I32Add);
            state.push_operator(Operator::GlobalSet { global_index: g });
        }

        state.push_operator(op);
        Ok(())
    }
}

// ── Opcode cost function ────────────────────────────────────────────
// Matches Nitro's pricing_v1 exactly for consensus compatibility.

#[rustfmt::skip]
fn opcode_ink_cost(op: &Operator) -> u64 {
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
        CallIndirect { .. } => 13610,

        // Unsupported opcodes: infinite cost rejects them at activation.
        _ => u64::MAX,
    }
}
