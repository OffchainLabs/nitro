use digest::Digest;
use sha3::Keccak256;

use crate::utils::Bytes32;

#[derive(Clone, Copy, PartialEq, Eq, Debug, Hash)]
#[repr(u8)]
pub enum ValueType {
    I32,
    I64,
    F32,
    F64,
    RefNull,
    FuncRef,
    InternalRef,
    StackBoundary,
}

impl ValueType {
    pub fn serialize(self) -> u8 {
        self as u8
    }
}

#[derive(Clone, Copy, PartialEq, Eq, Debug, Hash)]
pub enum IntegerValType {
    I32,
    I64,
}

impl Into<ValueType> for IntegerValType {
    fn into(self) -> ValueType {
        match self {
            IntegerValType::I32 => ValueType::I32,
            IntegerValType::I64 => ValueType::I64,
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct ProgramCounter {
    pub module: usize,
    pub func: usize,
    pub inst: usize,
    pub block_depth: usize,
}

impl ProgramCounter {
    pub fn new(module: usize, func: usize, inst: usize, block_depth: usize) -> ProgramCounter {
        ProgramCounter { module, func, inst, block_depth }
    }

    pub fn serialize(self) -> Bytes32 {
        let mut b = [0u8; 32];
        b[28..].copy_from_slice(&(self.inst as u32).to_be_bytes());
        b[24..28].copy_from_slice(&(self.func as u32).to_be_bytes());
        b[20..24].copy_from_slice(&(self.module as u32).to_be_bytes());
        Bytes32(b)
    }
}

#[derive(Clone, Copy, Debug)]
pub enum Value {
    I32(u32),
    I64(u64),
    F32(f32),
    F64(f64),
    RefNull,
    FuncRef(u32),
    InternalRef(ProgramCounter),
    StackBoundary,
}

impl Value {
    pub fn ty(self) -> ValueType {
        match self {
            Value::I32(_) => ValueType::I32,
            Value::I64(_) => ValueType::I64,
            Value::F32(_) => ValueType::F32,
            Value::F64(_) => ValueType::F64,
            Value::RefNull => ValueType::RefNull,
            Value::FuncRef(_) => ValueType::FuncRef,
            Value::InternalRef(_) => ValueType::InternalRef,
            Value::StackBoundary => ValueType::StackBoundary,
        }
    }

    pub fn contents_for_proof(self) -> Bytes32 {
        match self {
            Value::I32(x) => x.into(),
            Value::I64(x) => x.into(),
            Value::F32(x) => x.to_bits().into(),
            Value::F64(x) => x.to_bits().into(),
            Value::RefNull => Bytes32::default(),
            Value::FuncRef(x) => x.into(),
            Value::InternalRef(pc) => pc.serialize(),
            Value::StackBoundary => Bytes32::default(),
        }
    }

    pub fn serialize_for_proof(self) -> [u8; 33] {
        let mut ret = [0u8; 33];
        ret[0] = self.ty().serialize();
        ret[1..].copy_from_slice(&*self.contents_for_proof());
        ret
    }

    pub fn is_i32_zero(self) -> bool {
        match self {
            Value::I32(0) => true,
            Value::I32(_) => false,
            _ => panic!(
                "WASM validation failed: i32.eqz equivalent called on {:?}",
                self,
            ),
        }
    }

    pub fn is_i64_zero(self) -> bool {
        match self {
            Value::I64(0) => true,
            Value::I64(_) => false,
            _ => panic!(
                "WASM validation failed: i64.eqz equivalent called on {:?}",
                self,
            ),
        }
    }

    pub fn assume_u32(self) -> u32 {
        match self {
            Value::I32(x) => x,
            _ => panic!("WASM validation failed: assume_u32 called on {:?}", self),
        }
    }

    pub fn assume_u64(self) -> u64 {
        match self {
            Value::I64(x) => x,
            _ => panic!("WASM validation failed: assume_u64 called on {:?}", self),
        }
    }

    pub fn hash(self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update(b"Value:");
        h.update(&[self.ty() as u8]);
        h.update(self.contents_for_proof());
        h.finalize().into()
    }

    pub fn default_of_type(ty: ValueType) -> Value {
        match ty {
            ValueType::I32 => Value::I32(0),
            ValueType::I64 => Value::I64(0),
            ValueType::F32 => Value::F32(0.),
            ValueType::F64 => Value::F64(0.),
            ValueType::RefNull | ValueType::FuncRef | ValueType::InternalRef => Value::RefNull,
            ValueType::StackBoundary => panic!("Attempted to make default of StackBoundary type"),
        }
    }
}

impl PartialEq for Value {
    fn eq(&self, other: &Self) -> bool {
        self.ty() == other.ty() && self.contents_for_proof() == other.contents_for_proof()
    }
}

impl Eq for Value {}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct FunctionType {
    pub inputs: Vec<ValueType>,
    pub outputs: Vec<ValueType>,
}

impl FunctionType {
    pub fn new(inputs: Vec<ValueType>, outputs: Vec<ValueType>) -> FunctionType {
        FunctionType { inputs, outputs }
    }

    pub fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update(b"Function type:");
        h.update(Bytes32::from(self.inputs.len()));
        for input in &self.inputs {
            h.update(&[*input as u8]);
        }
        h.update(Bytes32::from(self.outputs.len()));
        for output in &self.outputs {
            h.update(&[*output as u8]);
        }
        h.finalize().into()
    }
}
