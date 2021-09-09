use crate::{
    binary::{WasmBinary, WasmSection},
    lir::Instruction,
    lir::Opcode,
    merkle::{Merkle, MerkleType},
    utils::{usize_to_u256_bytes, Bytes32},
    value::{Value, ValueType},
};
use digest::Digest;
use eyre::Result;
use sha3::Keccak256;

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
                inst.proving_argument_data = Some(dbg!(hashes[code_len - end_pc]));
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
    fn new(mut code: Vec<Instruction>, local_types: Vec<ValueType>) -> Function {
        let code_hashes = compute_hashes(&mut code);
        Function {
            code,
            code_hashes,
            local_types,
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

impl Machine {
    pub fn from_binary(bin: WasmBinary) -> Result<Machine> {
        let mut code = Vec::new();
        let mut globals = Vec::new();
        let mut types = Vec::new();
        let mut func_types = Vec::new();
        let mut start = 0;
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
                        .map(|(idx, c)| {
                            let func_ty = &func_types[idx];
                            let locals_with_params: Vec<ValueType> =
                                func_ty.inputs.iter().cloned().chain(c.locals).collect();
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
                                proving_argument_data: Some(
                                    Merkle::new(MerkleType::Value, empty_local_hashes).root(),
                                ),
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
                            for hir_inst in c.expr {
                                Instruction::extend_from_hir(
                                    &mut insts,
                                    func_ty.outputs.len(),
                                    hir_inst,
                                );
                            }
                            Instruction::extend_from_hir(
                                &mut insts,
                                func_ty.outputs.len(),
                                crate::binary::HirInstruction::Simple(Opcode::Return),
                            );
                            Function::new(insts, locals_with_params)
                        })
                        .collect();
                }
                WasmSection::Start(s) => {
                    assert!(start == 0, "Duplicate start section");
                    start = s as usize;
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
                WasmSection::Custom(_) => {}
            }
        }
        assert!(!code.is_empty());
        Ok(Machine {
            value_stack: vec![Value::RefNull],
            internal_stack: Vec::new(),
            block_stack: Vec::new(),
            frame_stack: Vec::new(),
            globals,
            funcs_merkle: Merkle::new(
                MerkleType::Function,
                code.iter().map(|f| f.code_hashes[0]).collect(),
            ),
            funcs: code,
            pc: (start, 0),
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
        h.update(self.funcs_merkle.root());
        h.finalize().into()
    }

    pub fn step(&mut self) {
        if self.halted {
            return;
        }

        let func = &self.funcs[self.pc.0];
        let code = &func.code;
        if code.len() <= self.pc.1 {
            eprintln!("Warning: ran off end of function");
            self.halted = true;
            return;
        }
        let inst = code[self.pc.1];
        dbg!(inst.opcode);
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
            Opcode::I32Add => {
                let a = self.value_stack.pop();
                let b = self.value_stack.pop();
                if let (Some(Value::I32(a)), Some(Value::I32(b))) = (a, b) {
                    self.value_stack.push(Value::I32(a.wrapping_add(b)));
                } else {
                    panic!("WASM validation failed: wrong types for i32.add");
                }
            }
            Opcode::I64Add => {
                let a = self.value_stack.pop();
                let b = self.value_stack.pop();
                if let (Some(Value::I64(a)), Some(Value::I64(b))) = (a, b) {
                    self.value_stack.push(Value::I64(a.wrapping_add(b)));
                } else {
                    panic!("WASM validation failed: wrong types for i64.add");
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
                self.value_stack.push(Value::I32((val == Value::StackBoundary) as u32));
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
        data.extend(usize_to_u256_bytes(
            self.value_stack.len() - unproven_stack_depth,
        ));
        for val in &self.value_stack[unproven_stack_depth..] {
            data.extend(val.serialize_for_proof());
        }

        let unproven_internal_stack_depth = self.internal_stack.len().saturating_sub(1);
        data.extend(hash_value_stack(
            &self.internal_stack[..unproven_internal_stack_depth],
        ));
        data.extend(usize_to_u256_bytes(
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

        data.extend(self.funcs_merkle.root());

        if let Some(next_inst) = func.code.get(self.pc.1) {
            if matches!(next_inst.opcode, Opcode::LocalGet | Opcode::LocalSet) {
                let locals = &self.frame_stack.last().unwrap().locals;
                let idx = next_inst.argument_data as usize;
                data.extend(locals[idx].serialize_for_proof());
                let locals_merkle =
                    Merkle::new(MerkleType::Value, locals.iter().map(|v| v.hash()).collect());
                data.extend(locals_merkle.prove(idx));
            } else if matches!(next_inst.opcode, Opcode::GlobalGet | Opcode::GlobalSet) {
                let idx = next_inst.argument_data as usize;
                data.extend(self.globals[idx].serialize_for_proof());
                let locals_merkle = Merkle::new(
                    MerkleType::Value,
                    self.globals.iter().map(|v| v.hash()).collect(),
                );
                data.extend(locals_merkle.prove(idx));
            } else if next_inst.opcode == Opcode::Call {
                let idx = next_inst.argument_data as usize;
                data.extend(self.funcs[idx].code_hashes[1]);
                data.extend(self.funcs[idx].code[0].serialize_for_proof());
                data.extend(self.funcs_merkle.prove(idx));
            }
        }

        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        &self.value_stack
    }
}
