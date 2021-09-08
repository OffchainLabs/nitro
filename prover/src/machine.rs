use crate::{
    binary::{HirInstruction, WasmBinary, WasmSection},
    lir::Instruction,
    lir::Opcode,
    merkle::Merkle,
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
        if inst.opcode == Opcode::Block {
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
        h.update(Merkle::new(self.locals.iter().map(|v| v.hash()).collect()).root());
        h.finalize().into()
    }

    fn serialize_for_proof(&self) -> [u8; 41] {
        let mut data = [0u8; 41];
        data[..9].copy_from_slice(&self.return_ref.serialize());
        data[9..]
            .copy_from_slice(&*Merkle::new(self.locals.iter().map(|v| v.hash()).collect()).root());
        data
    }
}

#[derive(Clone, Debug)]
pub struct Machine {
    value_stack: Vec<Value>,
    block_stack: Vec<usize>,
    frame_stack: Vec<StackFrame>,
    globals: Vec<Value>,
    funcs: Vec<Function>,
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

fn hash_block_stack(pcs: &[usize], hashes: &[Bytes32]) -> Bytes32 {
    hash_stack(pcs.iter().map(|&pc| hashes[pc]), "Bytes32 stack:")
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
        let mut start = 0;
        for sect in bin.sections {
            match sect {
                WasmSection::Code(sect_code) => {
                    if !code.is_empty() {
                        panic!("Duplicate code section");
                    }
                    code = sect_code
                        .into_iter()
                        .enumerate()
                        .map(|(idx, c)| {
                            let mut insts = Vec::new();
                            let empty_local_hashes = c
                                .locals
                                .iter()
                                .cloned()
                                .map(Value::default_of_type)
                                .map(Value::hash)
                                .collect::<Vec<_>>();
                            insts.push(Instruction {
                                opcode: Opcode::InitFrame,
                                argument_data: idx as u64,
                                proving_argument_data: Some(Merkle::new(empty_local_hashes).root()),
                            });
                            for hir_inst in c.expr {
                                Instruction::extend_from_hir(&mut insts, hir_inst);
                            }
                            Function::new(insts, c.locals)
                        })
                        .collect();
                }
                WasmSection::Start(s) => {
                    start = s as usize;
                }
                WasmSection::Globals(g) => {
                    if !globals.is_empty() {
                        panic!("Duplicate globals section");
                    }
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
                WasmSection::Custom(_) | WasmSection::Types(_) | WasmSection::Functions(_) => {}
            }
        }
        assert!(!code.is_empty());
        Ok(Machine {
            value_stack: vec![Value::RefNull],
            block_stack: Vec::new(),
            frame_stack: Vec::new(),
            globals,
            funcs: code,
            pc: (start, 0),
            halted: false,
        })
    }

    pub fn hash(&self) -> Bytes32 {
        if self.halted {
            return Bytes32::default();
        }
        // TODO: hash in functions so they can be jumped to
        let mut h = Keccak256::new();
        h.update(b"Machine:");
        h.update(&hash_value_stack(&self.value_stack));
        h.update(&hash_block_stack(
            &self.block_stack,
            &self.funcs[self.pc.0].code_hashes,
        ));
        h.update(hash_stack_frame_stack(&self.frame_stack));
        h.update(&self.funcs[self.pc.0].code_hashes[self.pc.1]);
        h.update(Merkle::new(self.globals.iter().map(|v| v.hash()).collect()).root());
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
                self.block_stack.push(inst.argument_data as usize);
            }
            Opcode::EndBlock => {
                self.block_stack.pop();
            }
            Opcode::EndBlockIf => {
                let x = self.value_stack.last().unwrap();
                if x.contents() != 0 {
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
            Opcode::Branch => {
                self.pc.1 = self.block_stack.pop().unwrap();
            }
            Opcode::BranchIf => {
                let x = self.value_stack.pop().unwrap();
                if x.contents() != 0 {
                    self.pc.1 = self.block_stack.pop().unwrap();
                }
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
                self.value_stack
                    .push(Value::I32((val.contents() == 0) as u32));
            }
            Opcode::Drop => {
                self.value_stack.pop().unwrap();
            }
            Opcode::I32Add | Opcode::I64Add => {
                let a = self.value_stack.pop().unwrap();
                let b = self.value_stack.pop().unwrap();
                let new = a.contents().wrapping_add(b.contents());
                if inst.opcode == Opcode::I32Add {
                    self.value_stack.push(Value::I32(new as u32));
                } else {
                    self.value_stack.push(Value::I64(new));
                }
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
            data.extend(val.serialize());
        }

        let func = &self.funcs[self.pc.0];
        data.extend(prove_window(
            &self.block_stack,
            |s| hash_block_stack(s, &func.code_hashes),
            |&pc| func.code_hashes[pc],
        ));

        data.extend(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        if self.pc.1 >= func.code.len() {
            data.extend(Bytes32::default());
            data.push(0);
        } else {
            data.extend(func.code_hashes[self.pc.1 + 1]);
            data.push(1);
            data.extend(func.code[self.pc.1].serialize_for_proof());
        }

        data.extend(Merkle::new(self.globals.iter().map(|v| v.hash()).collect()).root());

        if let Some(next_inst) = func.code.get(self.pc.1) {
            if next_inst.opcode == Opcode::LocalGet || next_inst.opcode == Opcode::LocalSet {
                let locals = &self.frame_stack.last().unwrap().locals;
                let idx = next_inst.argument_data as usize;
                data.extend(locals[idx].serialize());
                let locals_merkle = Merkle::new(locals.iter().map(|v| v.hash()).collect());
                data.extend(locals_merkle.prove(idx));
            } else if next_inst.opcode == Opcode::GlobalGet || next_inst.opcode == Opcode::GlobalSet
            {
                let idx = next_inst.argument_data as usize;
                data.extend(self.globals[idx].serialize());
                let locals_merkle = Merkle::new(self.globals.iter().map(|v| v.hash()).collect());
                data.extend(locals_merkle.prove(idx));
            }
        }

        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        &self.value_stack
    }
}
