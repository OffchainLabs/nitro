use crate::{
    binary::{Code, ExportKind, FunctionType, HirInstruction, WasmBinary, WasmSection},
    lir::Instruction,
    lir::{IBinOpType, Opcode},
    memory::Memory,
    merkle::{Merkle, MerkleType},
    utils::Bytes32,
    value::{IntegerValType, Value, ValueType},
};
use digest::Digest;
use eyre::Result;
use num_traits;
use sha3::Keccak256;
use std::convert::TryFrom;

#[derive(Clone, Debug)]
struct Function {
    code: Vec<Instruction>,
    code_hashes: Vec<Bytes32>,
    local_types: Vec<ValueType>,
}

fn compute_hashes(code: &mut Vec<Instruction>) -> Vec<Bytes32> {
    let mut prev_hash = Bytes32::default();
    let mut hashes = vec![prev_hash];
    let code_len = code.len();
    for inst in code.iter_mut().rev() {
        if inst.opcode == Opcode::Block || inst.opcode == Opcode::ArbitraryJumpIf {
            let end_pc = inst.argument_data as usize;
            if code_len - end_pc < hashes.len() {
                inst.proving_argument_data = Some(hashes[code_len - end_pc]);
            } else if code_len - end_pc == hashes.len() {
                inst.proving_argument_data = Some(Bytes32::default());
            } else {
                panic!("Block has backwards exit");
            }
        }
        let mut h = Keccak256::new();
        h.update("Instruction stack:");
        h.update(&inst.hash());
        h.update(&prev_hash);
        let hash = h.finalize().into();
        hashes.push(hash);
        prev_hash = hash;
    }
    hashes.reverse();
    hashes
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
        for hir_inst in code.expr {
            Instruction::extend_from_hir(&mut insts, func_ty.outputs.len(), hir_inst);
        }
        Instruction::extend_from_hir(
            &mut insts,
            func_ty.outputs.len(),
            crate::binary::HirInstruction::Simple(Opcode::Return),
        );
        let code_hashes = compute_hashes(&mut insts);
        Function {
            code: insts,
            code_hashes,
            local_types: locals_with_params,
        }
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
pub struct Machine {
    value_stack: Vec<Value>,
    internal_stack: Vec<Value>,
    block_stack: Vec<(usize, Bytes32)>,
    frame_stack: Vec<StackFrame>,
    globals: Vec<Value>,
    memory: Memory,
    funcs: Vec<Function>,
    funcs_merkle: Merkle,
    pc: (usize, usize),
    halted: bool,
}

fn hash_stack<I>(stack: I, prefix: &str) -> Bytes32
where
    I: IntoIterator<Item = Bytes32>,
{
    let mut hash = Bytes32::default();
    for item_hash in stack.into_iter() {
        let mut h = Keccak256::new();
        h.update(prefix);
        h.update(item_hash);
        h.update(&hash);
        hash = h.finalize().into();
    }
    hash
}

fn hash_value_stack(stack: &[Value]) -> Bytes32 {
    hash_stack(stack.iter().map(|v| v.hash()), "Value stack:")
}

fn hash_block_stack(pcs: &[(usize, Bytes32)]) -> Bytes32 {
    hash_stack(pcs.iter().map(|(_, h)| *h), "Bytes32 stack:")
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
fn exec_ibin_op<T: num_traits::WrappingAdd + num_traits::WrappingMul + num_traits::WrappingSub>(
    a: &T,
    b: &T,
    op: &IBinOpType,
) -> T {
    match op {
        IBinOpType::Add => return a.wrapping_add(b),
        IBinOpType::Sub => return a.wrapping_sub(b),
        IBinOpType::Mul => return a.wrapping_mul(b),
    }
}

impl Machine {
    pub fn from_binary(bin: WasmBinary) -> Result<Machine> {
        let mut code = Vec::new();
        let mut globals = Vec::new();
        let mut types = Vec::new();
        let mut func_types = Vec::new();
        let mut start = None;
        let mut memory = Memory::default();
        let mut main = None;
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
                WasmSection::Custom(_) => {}
            }
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
        Ok(Machine {
            value_stack: vec![Value::RefNull],
            internal_stack: Vec::new(),
            block_stack: Vec::new(),
            frame_stack: Vec::new(),
            memory,
            globals,
            funcs_merkle: Merkle::new(
                MerkleType::Function,
                code.iter().map(|f| f.code_hashes[0]).collect(),
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
        h.update(&hash_block_stack(&self.block_stack));
        h.update(hash_stack_frame_stack(&self.frame_stack));
        h.update(&self.funcs[self.pc.0].code_hashes[self.pc.1]);
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
                self.block_stack.push((idx, func.code_hashes[idx]));
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
                self.pc.1 = self.block_stack.pop().unwrap().0;
            }
            Opcode::BranchIf => {
                let x = self.value_stack.pop().unwrap();
                if !x.is_i32_zero() {
                    self.pc.1 = self.block_stack.pop().unwrap().0;
                }
            }
            Opcode::Return => {
                let frame = self.frame_stack.pop().unwrap();
                match frame.return_ref {
                    Value::RefNull => {
                        self.halted = true;
                    }
                    Value::Ref(pc, _) => self.pc = pc,
                    v => panic!("Attempted to return into an invalid reference: {:?}", v),
                }
            }
            Opcode::Call => {
                self.value_stack
                    .push(Value::Ref(self.pc, func.code_hashes[self.pc.1]));
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
            Opcode::I32Eqz => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack.push(Value::I32(val.is_i32_zero() as u32));
            }
            Opcode::Drop => {
                self.value_stack.pop().unwrap();
            }
            Opcode::IBinOp(w, op) => {
                let vb = self.value_stack.pop();
                let va = self.value_stack.pop();
                match w {
                    IntegerValType::I32 => {
                        if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                            self.value_stack.push(Value::I32(exec_ibin_op(&a, &b, &op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i32binop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                            self.value_stack.push(Value::I64(exec_ibin_op(&a, &b, &op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i64binop");
                        }
                    }
                }
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
        }
    }

    pub fn is_halted(&self) -> bool {
        self.halted
    }

    pub fn serialize_proof(&self) -> Vec<u8> {
        // Could be variable, but not worth it yet
        const STACK_PROVING_DEPTH: usize = 2;

        let mut data = Vec::new();
        let unproven_stack_depth = self.value_stack.len().saturating_sub(STACK_PROVING_DEPTH);
        data.extend(hash_value_stack(&self.value_stack[..unproven_stack_depth]));
        data.extend(Bytes32::from(self.value_stack.len() - unproven_stack_depth));
        for val in &self.value_stack[unproven_stack_depth..] {
            data.extend(val.serialize_for_proof());
        }

        let unproven_internal_stack_depth = self.internal_stack.len().saturating_sub(1);
        data.extend(hash_value_stack(
            &self.internal_stack[..unproven_internal_stack_depth],
        ));
        data.extend(Bytes32::from(
            self.internal_stack.len() - unproven_internal_stack_depth,
        ));
        for val in &self.internal_stack[unproven_internal_stack_depth..] {
            data.extend(val.serialize_for_proof());
        }

        let func = &self.funcs[self.pc.0];
        data.extend(prove_window(
            &self.block_stack,
            |s| hash_block_stack(s),
            |(_, h)| *h,
        ));

        data.extend(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        data.extend(func.code_hashes[self.pc.1 + 1]);
        data.extend(func.code[self.pc.1].serialize_for_proof());

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
            } else if next_inst.opcode == Opcode::Call {
                let idx = next_inst.argument_data as usize;
                data.extend(self.funcs[idx].code_hashes[1]);
                data.extend(self.funcs[idx].code[0].serialize_for_proof());
                data.extend(
                    self.funcs_merkle
                        .prove(idx)
                        .expect("Out of bounds function access"),
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
                    println!("{:?}", mem_merkle.prove(idx));
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
