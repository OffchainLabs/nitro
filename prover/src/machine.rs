use crate::{
    binary::{Code, ElementMode, ExportKind, HirInstruction, TableType, WasmBinary, WasmSection},
    lir::Instruction,
    lir::{FunctionCodegenState, IBinOpType, IRelOpType, IUnOpType, Opcode},
    memory::Memory,
    merkle::{Merkle, MerkleType},
    reinterpret::{ReinterpretAsSigned, ReinterpretAsUnsigned},
    utils::Bytes32,
    value::{FunctionType, IntegerValType, Value, ValueType},
};
use digest::Digest;
use eyre::Result;
use num::{traits::PrimInt, Zero};
use sha3::Keccak256;
use std::{convert::TryFrom, num::Wrapping};

#[derive(Clone, Debug)]
struct Function {
    code: Vec<Instruction>,
    code_merkle: Merkle,
    local_types: Vec<ValueType>,
}

impl Function {
    fn new(code: Code, func_ty: &FunctionType) -> Function {
        let locals_with_params: Vec<ValueType> =
            func_ty.inputs.iter().cloned().chain(code.locals).collect();
        let mut insts = Vec::new();
        let empty_local_hashes = locals_with_params
            .iter()
            .cloned()
            .map(Value::default_of_type)
            .map(Value::hash)
            .collect::<Vec<_>>();
        insts.push(Instruction {
            opcode: Opcode::InitFrame,
            argument_data: 0,
            proving_argument_data: Some(Merkle::new(MerkleType::Value, empty_local_hashes).root()),
        });
        // Fill in parameters
        for i in (0..func_ty.inputs.len()).rev() {
            insts.push(Instruction {
                opcode: Opcode::LocalSet,
                argument_data: i as u64,
                proving_argument_data: None,
            });
        }
        insts.push(Instruction::simple(Opcode::PushStackBoundary));
        let codegen_state = FunctionCodegenState::new(func_ty.outputs.len());
        for hir_inst in code.expr {
            Instruction::extend_from_hir(&mut insts, codegen_state, hir_inst);
        }
        Instruction::extend_from_hir(
            &mut insts,
            codegen_state,
            crate::binary::HirInstruction::Simple(Opcode::Return),
        );
        let code_merkle = Merkle::new(
            MerkleType::Instruction,
            insts.iter().map(|i| i.hash()).collect(),
        );
        Function {
            code: insts,
            code_merkle,
            local_types: locals_with_params,
        }
    }

    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Function:");
        h.update(self.code_merkle.root());
        h.finalize().into()
    }
}

#[derive(Clone, Debug)]
struct StackFrame {
    return_ref: Value,
    locals: Vec<Value>,
}

impl StackFrame {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Stack frame:");
        h.update(&self.return_ref.hash());
        h.update(
            Merkle::new(
                MerkleType::Value,
                self.locals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        h.finalize().into()
    }

    fn serialize_for_proof(&self) -> Vec<u8> {
        let mut data = Vec::new();
        data.extend(self.return_ref.serialize_for_proof());
        data.extend(
            Merkle::new(
                MerkleType::Value,
                self.locals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        data
    }
}

#[derive(Clone, Debug)]
struct TableElement {
    func_ty: FunctionType,
    val: Value,
}

impl Default for TableElement {
    fn default() -> Self {
        TableElement {
            func_ty: FunctionType::default(),
            val: Value::RefNull,
        }
    }
}

impl TableElement {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Table element:");
        h.update(self.func_ty.hash());
        h.update(self.val.hash());
        h.finalize().into()
    }
}

#[derive(Clone, Debug)]
struct Table {
    ty: TableType,
    elems: Vec<TableElement>,
}

fn hash_table(ty: TableType, elements_root: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update("Table:");
    h.update(&[Into::<ValueType>::into(ty.ty).serialize()]);
    h.update(ty.limits.minimum_size.to_be_bytes());
    h.update(ty.limits.maximum_size.unwrap_or(u32::MAX).to_be_bytes());
    h.update(elements_root);
    h.finalize().into()
}

#[derive(Clone, Debug)]
pub struct Machine {
    value_stack: Vec<Value>,
    internal_stack: Vec<Value>,
    block_stack: Vec<usize>,
    frame_stack: Vec<StackFrame>,
    globals: Vec<Value>,
    memory: Memory,
    tables: Vec<Table>,
    tables_merkle: Merkle,
    tables_elements_merkles: Vec<Merkle>,
    funcs: Vec<Function>,
    funcs_merkle: Merkle,
    pc: (usize, usize),
    halted: bool,
}

fn hash_stack<I, D>(stack: I, prefix: &str) -> Bytes32
where
    I: IntoIterator<Item = D>,
    D: AsRef<[u8]>,
{
    let mut hash = Bytes32::default();
    for item in stack.into_iter() {
        let mut h = Keccak256::new();
        h.update(prefix);
        h.update(item.as_ref());
        h.update(&hash);
        hash = h.finalize().into();
    }
    hash
}

fn hash_value_stack(stack: &[Value]) -> Bytes32 {
    hash_stack(stack.iter().map(|v| v.hash()), "Value stack:")
}

fn hash_pc_stack(pcs: &[usize]) -> Bytes32 {
    hash_stack(
        pcs.iter().map(|pc| (*pc as u64).to_be_bytes()),
        "Program counter stack:",
    )
}

fn hash_stack_frame_stack(frames: &[StackFrame]) -> Bytes32 {
    hash_stack(frames.iter().map(|f| f.hash()), "Stack frame stack:")
}

#[must_use]
fn prove_window<T, F, D, G>(items: &[T], stack_hasher: F, encoder: G) -> Vec<u8>
where
    F: Fn(&[T]) -> Bytes32,
    D: AsRef<[u8]>,
    G: Fn(&T) -> D,
{
    let mut data = Vec::with_capacity(33);
    if items.is_empty() {
        data.extend(Bytes32::default());
        data.push(0);
    } else {
        let last_idx = items.len() - 1;
        data.extend(stack_hasher(&items[..last_idx]));
        data.push(1);
        data.extend(encoder(&items[last_idx]).as_ref());
    }
    data
}

#[must_use]
fn prove_stack<T, F, D, G>(
    items: &[T],
    proving_depth: usize,
    stack_hasher: F,
    encoder: G,
) -> Vec<u8>
where
    F: Fn(&[T]) -> Bytes32,
    D: AsRef<[u8]>,
    G: Fn(&T) -> D,
{
    let mut data = Vec::with_capacity(33);
    let unproven_stack_depth = items.len().saturating_sub(proving_depth);
    data.extend(stack_hasher(&items[..unproven_stack_depth]));
    data.extend(Bytes32::from(items.len() - unproven_stack_depth));
    for val in &items[unproven_stack_depth..] {
        data.extend(encoder(val).as_ref());
    }
    data
}

#[must_use]
fn exec_ibin_op<T>(a: T, b: T, op: IBinOpType) -> T
where
    Wrapping<T>: ReinterpretAsSigned,
    T: Zero,
{
    let a = Wrapping(a);
    let b = Wrapping(b);
    if matches!(
        op,
        IBinOpType::DivS | IBinOpType::DivU | IBinOpType::RemS | IBinOpType::RemU,
    ) {
        if b.is_zero() {
            return T::zero();
        }
    }
    let res = match op {
        IBinOpType::Add => a + b,
        IBinOpType::Sub => a - b,
        IBinOpType::Mul => a * b,
        IBinOpType::DivS => (a.cast_signed() / b.cast_signed()).cast_unsigned(),
        IBinOpType::DivU => a / b,
        IBinOpType::RemS => (a.cast_signed() % b.cast_signed()).cast_unsigned(),
        IBinOpType::RemU => a % b,
        IBinOpType::And => a & b,
        IBinOpType::Or => a | b,
        IBinOpType::Xor => a ^ b,
        IBinOpType::Shl => a << b.cast_usize(),
        IBinOpType::ShrS => (a.cast_signed() >> b.cast_usize()).cast_unsigned(),
        IBinOpType::ShrU => a >> b.cast_usize(),
        IBinOpType::Rotl => a.rotl(b.cast_usize()),
        IBinOpType::Rotr => a.rotr(b.cast_usize()),
    };
    res.0
}

#[must_use]
fn exec_iun_op<T>(a: T, op: IUnOpType) -> u32
where
    T: PrimInt,
{
    match op {
        IUnOpType::Clz => a.leading_zeros(),
        IUnOpType::Ctz => a.trailing_zeros(),
        IUnOpType::Popcnt => a.count_ones(),
    }
}

fn exec_irel_op<T>(a: T, b: T, op: IRelOpType) -> Value
where
    T: Ord,
{
    let res = match op {
        IRelOpType::Eq => a == b,
        IRelOpType::Ne => a != b,
        IRelOpType::Lt => a < b,
        IRelOpType::Gt => a > b,
        IRelOpType::Le => a <= b,
        IRelOpType::Ge => a >= b,
    };
    Value::I32(res as u32)
}

impl Machine {
    pub fn from_binary(bin: WasmBinary, always_merkelize_memory: bool) -> Result<Machine> {
        let mut code = Vec::new();
        let mut globals = Vec::new();
        let mut types = Vec::new();
        let mut func_types = Vec::new();
        let mut start = None;
        let mut memory = Memory::default();
        let mut main = None;
        let mut tables = Vec::new();
        for sect in bin.sections {
            match sect {
                WasmSection::Types(t) => {
                    assert!(types.is_empty(), "Duplicate types section");
                    types = t;
                }
                WasmSection::Functions(f) => {
                    assert!(func_types.is_empty(), "Duplicate types section");
                    func_types = f.into_iter().map(|x| types[x as usize].clone()).collect();
                }
                WasmSection::Code(sect_code) => {
                    assert!(code.is_empty(), "Duplicate code section");
                    code = sect_code
                        .into_iter()
                        .enumerate()
                        .map(|(idx, c)| Function::new(c, &func_types[idx]))
                        .collect();
                }
                WasmSection::Start(s) => {
                    assert!(start.is_none(), "Duplicate start section");
                    start = Some(s);
                }
                WasmSection::Memories(m) => {
                    assert!(memory.size() == 0, "Duplicate memories section");
                    assert!(m.len() <= 1, "Multiple memories are not supported");
                    if let Some(limits) = m.get(0) {
                        // We ignore the maximum size
                        let size = usize::try_from(limits.minimum_size)
                            .ok()
                            .and_then(|x| x.checked_mul(Memory::PAGE_SIZE))
                            .expect("Memory size is too large");
                        memory = Memory::new(size);
                    }
                }
                WasmSection::Globals(g) => {
                    assert!(globals.is_empty(), "Duplicate globals section");
                    globals = g
                        .into_iter()
                        .map(|g| {
                            if let [insn] = g.initializer.as_slice() {
                                if let Some(val) = insn.get_const_output() {
                                    return val;
                                }
                            }
                            panic!("Global initializer isn't a constant");
                        })
                        .collect();
                }
                WasmSection::Exports(exports) => {
                    for export in exports {
                        if export.name == "main" {
                            if let ExportKind::Function(idx) = export.kind {
                                main = Some(idx);
                            } else {
                                panic!("Got non-function export {:?} for main", export.kind);
                            }
                        }
                    }
                }
                WasmSection::Datas(datas) => {
                    for data in datas {
                        if let Some(loc) = data.active_location {
                            assert_eq!(loc.memory, 0, "Attempted to write to nonexistant memory");
                            let mut offset = None;
                            if let [insn] = loc.offset.as_slice() {
                                if let Some(Value::I32(x)) = insn.get_const_output() {
                                    offset = Some(x);
                                }
                            }
                            let offset = usize::try_from(
                                offset.expect("Non-constant data offset expression"),
                            )
                            .unwrap();
                            if !matches!(
                                offset.checked_add(data.data.len()),
                                Some(x) if (x as u64) < memory.size(),
                            ) {
                                panic!(
                                    "Out-of-bounds data memory init with offset {} and size {}",
                                    offset,
                                    data.data.len(),
                                );
                            }
                            memory.set_range(offset, &data.data);
                        }
                    }
                }
                WasmSection::Tables(t) => {
                    assert!(tables.is_empty(), "Duplicate tables section");
                    for table in t {
                        tables.push(Table {
                            elems: vec![
                                TableElement::default();
                                usize::try_from(table.limits.minimum_size).unwrap()
                            ],
                            ty: table,
                        });
                    }
                }
                WasmSection::Elements(elems) => {
                    for elem in elems {
                        if let ElementMode::Active(t, o) = elem.mode {
                            let mut offset = None;
                            if let [insn] = o.as_slice() {
                                if let Some(Value::I32(x)) = insn.get_const_output() {
                                    offset = Some(x);
                                }
                            }
                            let offset = usize::try_from(
                                offset.expect("Non-constant data offset expression"),
                            )
                            .unwrap();
                            let t = usize::try_from(t).unwrap();
                            assert_eq!(tables[t].ty.ty, elem.ty);
                            let contents: Vec<_> = elem
                                .init
                                .into_iter()
                                .map(|i| {
                                    let insn = match i.as_slice() {
                                        [x] => x,
                                        _ => panic!(
                                            "Element initializer isn't one instruction: {:?}",
                                            o
                                        ),
                                    };
                                    match insn.get_const_output() {
                                        Some(v @ Value::RefNull) => TableElement {
                                            func_ty: FunctionType::default(),
                                            val: v,
                                        },
                                        Some(Value::FuncRef(x)) => TableElement {
                                            func_ty: func_types[usize::try_from(x).unwrap()]
                                                .clone(),
                                            val: Value::FuncRef(x),
                                        },
                                        _ => panic!("Invalid element initializer {:?}", insn),
                                    }
                                })
                                .collect();
                            let len = contents.len();
                            tables[t].elems[offset..][..len].clone_from_slice(&contents);
                        }
                    }
                }
                WasmSection::Custom(_) | WasmSection::DataCount(_) => {}
            }
        }
        if always_merkelize_memory {
            memory.cache_merkle_tree();
        }
        let mut entrypoint = Vec::new();
        if let Some(s) = start {
            assert!(
                func_types[s as usize] == FunctionType::default(),
                "Start function takes inputs or outputs",
            );
            entrypoint.push(HirInstruction::WithIdx(Opcode::Call, s));
        }
        if let Some(m) = main {
            let mut expected_type = FunctionType::default();
            expected_type.inputs.push(ValueType::I32); // argc
            expected_type.inputs.push(ValueType::I32); // argv
            expected_type.outputs.push(ValueType::I32); // ret
            assert!(
                func_types[m as usize] == expected_type,
                "Main function doesn't match expected signature of [argc, argv] -> [ret]",
            );
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(HirInstruction::WithIdx(Opcode::Call, m));
            entrypoint.push(HirInstruction::Simple(Opcode::Drop));
        }
        let entrypoint_idx = code.len();
        code.push(Function::new(
            Code {
                locals: Vec::new(),
                expr: entrypoint,
            },
            &FunctionType::default(),
        ));
        let tables_elements_merkles: Vec<_> = tables
            .iter()
            .map(|t| {
                Merkle::new(
                    MerkleType::TableElement,
                    t.elems.iter().map(|e| e.hash()).collect(),
                )
            })
            .collect();
        Ok(Machine {
            value_stack: vec![Value::RefNull],
            internal_stack: Vec::new(),
            block_stack: Vec::new(),
            frame_stack: Vec::new(),
            memory,
            globals,
            tables_merkle: Merkle::new(
                MerkleType::Table,
                tables
                    .iter()
                    .zip(tables_elements_merkles.iter())
                    .map(|(t, m)| hash_table(t.ty, m.root()))
                    .collect(),
            ),
            tables_elements_merkles,
            tables,
            funcs_merkle: Merkle::new(
                MerkleType::Function,
                code.iter().map(|f| f.hash()).collect(),
            ),
            funcs: code,
            pc: (entrypoint_idx, 0),
            halted: false,
        })
    }

    pub fn hash(&self) -> Bytes32 {
        if self.halted {
            return Bytes32::default();
        }
        let mut h = Keccak256::new();
        h.update(b"Machine:");
        h.update(&hash_value_stack(&self.value_stack));
        h.update(&hash_value_stack(&self.internal_stack));
        h.update(&hash_pc_stack(&self.block_stack));
        h.update(hash_stack_frame_stack(&self.frame_stack));
        h.update(&(self.pc.0 as u64).to_be_bytes());
        h.update(&(self.pc.1 as u64).to_be_bytes());
        h.update(
            Merkle::new(
                MerkleType::Value,
                self.globals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        h.update(self.memory.hash());
        h.update(self.funcs_merkle.root());
        h.finalize().into()
    }

    pub fn get_next_instruction(&self) -> Option<Instruction> {
        if self.halted {
            return None;
        }
        self.funcs[self.pc.0].code.get(self.pc.1).cloned()
    }

    pub fn step(&mut self) {
        if self.halted {
            return;
        }

        let func = &self.funcs[self.pc.0];
        let code = &func.code;
        let inst = code[self.pc.1];
        self.pc.1 += 1;
        match inst.opcode {
            Opcode::Unreachable => {
                self.halted = true;
            }
            Opcode::Nop => {}
            Opcode::Block => {
                let idx = inst.argument_data as usize;
                self.block_stack.push(idx);
            }
            Opcode::EndBlock => {
                self.block_stack.pop();
            }
            Opcode::EndBlockIf => {
                let x = self.value_stack.last().unwrap();
                if !x.is_i32_zero() {
                    self.block_stack.pop().unwrap();
                }
            }
            Opcode::InitFrame => {
                let return_ref = self.value_stack.pop().unwrap();
                self.frame_stack.push(StackFrame {
                    return_ref,
                    locals: func
                        .local_types
                        .iter()
                        .cloned()
                        .map(Value::default_of_type)
                        .collect(),
                });
            }
            Opcode::ArbitraryJumpIf => {
                let x = self.value_stack.pop().unwrap();
                if !x.is_i32_zero() {
                    self.pc.1 = inst.argument_data as usize;
                }
            }
            Opcode::Branch => {
                self.pc.1 = self.block_stack.pop().unwrap();
            }
            Opcode::BranchIf => {
                let x = self.value_stack.pop().unwrap();
                if !x.is_i32_zero() {
                    self.pc.1 = self.block_stack.pop().unwrap();
                }
            }
            Opcode::Return => {
                let frame = self.frame_stack.pop().unwrap();
                match frame.return_ref {
                    Value::RefNull => {
                        self.halted = true;
                    }
                    Value::InternalRef(pc) => self.pc = pc,
                    v => panic!("Attempted to return into an invalid reference: {:?}", v),
                }
            }
            Opcode::Call => {
                self.value_stack.push(Value::InternalRef(self.pc));
                self.pc = (inst.argument_data as usize, 0);
            }
            Opcode::LocalGet => {
                let val = self.frame_stack.last().unwrap().locals[inst.argument_data as usize];
                self.value_stack.push(val);
            }
            Opcode::LocalSet => {
                let val = self.value_stack.pop().unwrap();
                self.frame_stack.last_mut().unwrap().locals[inst.argument_data as usize] = val;
            }
            Opcode::GlobalGet => {
                self.value_stack
                    .push(self.globals[inst.argument_data as usize]);
            }
            Opcode::GlobalSet => {
                let val = self.value_stack.pop().unwrap();
                self.globals[inst.argument_data as usize] = val;
            }
            Opcode::MemoryLoad { ty, bytes, signed } => {
                let base = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    x => panic!(
                        "WASM validation failed: top of stack before memory load is {:?}",
                        x,
                    ),
                };
                if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                    let val = self.memory.get_value(idx, ty, bytes, signed);
                    if let Some(val) = val {
                        self.value_stack.push(val);
                    } else {
                        self.halted = true;
                    }
                } else {
                    self.halted = true;
                }
            }
            Opcode::MemoryStore { ty: _, bytes } => {
                let val = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x.into(),
                    Some(Value::I64(x)) => x,
                    Some(Value::F32(x)) => x.to_bits().into(),
                    Some(Value::F64(x)) => x.to_bits(),
                    x => panic!(
                        "WASM validation failed: attempted to memory store type {:?}",
                        x,
                    ),
                };
                let base = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    x => panic!(
                        "WASM validation failed: attempted to memory store with index type {:?}",
                        x,
                    ),
                };
                if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                    if !self.memory.store_value(idx, val, bytes) {
                        self.halted = true;
                    }
                } else {
                    self.halted = true;
                }
            }
            Opcode::I32Const => {
                self.value_stack.push(Value::I32(inst.argument_data as u32));
            }
            Opcode::I64Const => {
                self.value_stack.push(Value::I64(inst.argument_data));
            }
            Opcode::F32Const => {
                self.value_stack
                    .push(Value::F32(f32::from_bits(inst.argument_data as u32)));
            }
            Opcode::F64Const => {
                self.value_stack
                    .push(Value::F64(f64::from_bits(inst.argument_data)));
            }
            Opcode::FuncRefConst => {
                self.value_stack
                    .push(Value::FuncRef(inst.argument_data as u32));
            }
            Opcode::I32Eqz => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack.push(Value::I32(val.is_i32_zero() as u32));
            }
            Opcode::I64Eqz => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack.push(Value::I32(val.is_i64_zero() as u32));
            }
            Opcode::IRelOp(t, op, signed) => {
                let vb = self.value_stack.pop();
                let va = self.value_stack.pop();
                match t {
                    IntegerValType::I32 => {
                        if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                            if signed {
                                self.value_stack.push(exec_irel_op(a as i32, b as i32, op));
                            } else {
                                self.value_stack.push(exec_irel_op(a, b, op));
                            }
                        } else {
                            panic!("WASM validation failed: wrong types for i32relop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                            if signed {
                                self.value_stack.push(exec_irel_op(a as i64, b as i64, op));
                            } else {
                                self.value_stack.push(exec_irel_op(a, b, op));
                            }
                        } else {
                            panic!("WASM validation failed: wrong types for i64relop");
                        }
                    }
                }
            }
            Opcode::Drop => {
                self.value_stack.pop().unwrap();
            }
            Opcode::Select => {
                let selector_zero = self.value_stack.pop().unwrap().is_i32_zero();
                let val2 = self.value_stack.pop().unwrap();
                let val1 = self.value_stack.pop().unwrap();
                if selector_zero {
                    self.value_stack.push(val2);
                } else {
                    self.value_stack.push(val1);
                }
            }
            Opcode::IUnOp(w, op) => {
                let va = self.value_stack.pop();
                match w {
                    IntegerValType::I32 => {
                        if let Some(Value::I32(a)) = va {
                            self.value_stack.push(Value::I32(exec_iun_op(a, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i32unop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let Some(Value::I64(a)) = va {
                            self.value_stack.push(Value::I64(exec_iun_op(a, op) as u64));
                        } else {
                            panic!("WASM validation failed: wrong types for i64unop");
                        }
                    }
                }
            }
            Opcode::IBinOp(w, op) => {
                let vb = self.value_stack.pop();
                let va = self.value_stack.pop();
                match w {
                    IntegerValType::I32 => {
                        if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                            self.value_stack.push(Value::I32(exec_ibin_op(a, b, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i32binop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                            self.value_stack.push(Value::I64(exec_ibin_op(a, b, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i64binop");
                        }
                    }
                }
            }
            Opcode::I32WrapI64 => {
                let x = match self.value_stack.pop() {
                    Some(Value::I64(x)) => x,
                    v => panic!(
                        "WASM validation failed: wrong type for i32.wrapi64: {:?}",
                        v,
                    ),
                };
                self.value_stack.push(Value::I32(x as u32));
            }
            Opcode::I64ExtendI32(signed) => {
                let x = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    v => panic!(
                        "WASM validation failed: wrong type for i64.extendi32: {:?}",
                        v,
                    ),
                };
                let x64 = match signed {
                    true => x as i32 as i64 as u64,
                    false => x as u32 as u64,
                };
                self.value_stack.push(Value::I64(x64));
            }
            Opcode::PushStackBoundary => {
                self.value_stack.push(Value::StackBoundary);
            }
            Opcode::MoveFromStackToInternal => {
                self.internal_stack.push(self.value_stack.pop().unwrap());
            }
            Opcode::MoveFromInternalToStack => {
                self.value_stack.push(self.internal_stack.pop().unwrap());
            }
            Opcode::IsStackBoundary => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack
                    .push(Value::I32((val == Value::StackBoundary) as u32));
            }
            Opcode::Dup => {
                let val = self.value_stack.last().cloned().unwrap();
                self.value_stack.push(val);
            }
        }
    }

    pub fn is_halted(&self) -> bool {
        self.halted
    }

    pub fn serialize_proof(&self) -> Vec<u8> {
        // Could be variable, but not worth it yet
        const STACK_PROVING_DEPTH: usize = 3;

        let mut data = Vec::new();

        data.extend(prove_stack(
            &self.value_stack,
            STACK_PROVING_DEPTH,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        data.extend(prove_stack(
            &self.internal_stack,
            1,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        data.extend(prove_stack(&self.block_stack, 1, hash_pc_stack, |pc| {
            (*pc as u64).to_be_bytes()
        }));

        data.extend(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        data.extend(&(self.pc.0 as u64).to_be_bytes());
        data.extend(&(self.pc.1 as u64).to_be_bytes());

        data.extend(
            Merkle::new(
                MerkleType::Value,
                self.globals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );

        let mem_merkle = self.memory.merkelize();
        data.extend((self.memory.size() as u64).to_be_bytes());
        data.extend(mem_merkle.root());

        data.extend(self.funcs_merkle.root());

        // End machine serialization, begin proof serialization

        let func = &self.funcs[self.pc.0];
        data.extend(func.code[self.pc.1].serialize_for_proof());
        data.extend(
            func.code_merkle
                .prove(self.pc.1)
                .expect("Failed to prove against code merkle"),
        );
        data.extend(
            self.funcs_merkle
                .prove(self.pc.0)
                .expect("Failed to prove against function merkle"),
        );

        if let Some(next_inst) = func.code.get(self.pc.1) {
            if matches!(next_inst.opcode, Opcode::LocalGet | Opcode::LocalSet) {
                let locals = &self.frame_stack.last().unwrap().locals;
                let idx = next_inst.argument_data as usize;
                data.extend(locals[idx].serialize_for_proof());
                let locals_merkle =
                    Merkle::new(MerkleType::Value, locals.iter().map(|v| v.hash()).collect());
                data.extend(
                    locals_merkle
                        .prove(idx)
                        .expect("Out of bounds local access"),
                );
            } else if matches!(next_inst.opcode, Opcode::GlobalGet | Opcode::GlobalSet) {
                let idx = next_inst.argument_data as usize;
                data.extend(self.globals[idx].serialize_for_proof());
                let locals_merkle = Merkle::new(
                    MerkleType::Value,
                    self.globals.iter().map(|v| v.hash()).collect(),
                );
                data.extend(
                    locals_merkle
                        .prove(idx)
                        .expect("Out of bounds global access"),
                );
            } else if matches!(
                next_inst.opcode,
                Opcode::MemoryLoad { .. } | Opcode::MemoryStore { .. }
            ) {
                let is_store = matches!(next_inst.opcode, Opcode::MemoryStore { .. });
                let stack_idx_offset = if is_store {
                    // The index is one item below the top stack item for a memory store
                    1
                } else {
                    0
                };
                let base = match self
                    .value_stack
                    .get(self.value_stack.len() - 1 - stack_idx_offset)
                {
                    Some(Value::I32(x)) => *x,
                    x => panic!("WASM validation failed: memory index type is {:?}", x),
                };
                if let Some(mut idx) = u64::from(base)
                    .checked_add(next_inst.argument_data)
                    .and_then(|x| usize::try_from(x).ok())
                {
                    // Prove the leaf this index is in, and the next one, if they are within the memory's size.
                    idx /= Memory::LEAF_SIZE;
                    data.extend(self.memory.get_leaf_data(idx));
                    data.extend(mem_merkle.prove(idx).unwrap_or_default());
                    // Now prove the next leaf too, in case it's accessed.
                    let next_leaf_idx = idx.saturating_add(1);
                    data.extend(self.memory.get_leaf_data(next_leaf_idx));
                    let second_mem_merkle = if is_store {
                        // For stores, prove the second merkle against a state after the first leaf is set.
                        // This state also happens to have the second leaf set, but that's irrelevant.
                        let mut copy = self.clone();
                        copy.step();
                        copy.memory.merkelize().into_owned()
                    } else {
                        mem_merkle.into_owned()
                    };
                    data.extend(second_mem_merkle.prove(next_leaf_idx).unwrap_or_default());
                }
            }
        }

        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        &self.value_stack
    }
}
