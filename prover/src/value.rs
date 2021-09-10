use digest::Digest;
use sha3::Keccak256;

use crate::utils::Bytes32;

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum IntegerValType {
    I32,
    I64,
}

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum ValueType {
    I32,
    I64,
    F32,
    F64,
    RefNull,
    FuncRef,
    ExternRef,
    StackBoundary,
}

impl ValueType {
    pub fn serialize(self) -> u8 {
        self as u8
    }
}

#[derive(Clone, Copy, Debug)]
pub enum Value {
    I32(u32),
    I64(u64),
    F32(f32),
    F64(f64),
    RefNull,
    Ref((usize, usize), Bytes32),
    #[allow(dead_code)]
    RefExtern(u32, Bytes32),
    StackBoundary,
}

impl Value {
    pub fn canonicalize(&mut self) {
        match self {
            Value::F32(x) if x.is_nan() => {
                *x = f32::from_bits(0b01111111110000000000000000000000_u32);
            }
            Value::F64(x) if x.is_nan() => {
                *x = f64::from_bits(
                    0b0111111111111000000000000000000000000000000000000000000000000000_u64,
                );
            }
            _ => {}
        }
    }

    pub fn ty(self) -> ValueType {
        match self {
            Value::I32(_) => ValueType::I32,
            Value::I64(_) => ValueType::I64,
            Value::F32(_) => ValueType::F32,
            Value::F64(_) => ValueType::F64,
            Value::RefNull => ValueType::RefNull,
            Value::Ref(_, _) => ValueType::FuncRef,
            Value::RefExtern(_, _) => ValueType::ExternRef,
            Value::StackBoundary => ValueType::StackBoundary,
        }
    }

    pub fn contents_for_proof(mut self) -> Bytes32 {
        self.canonicalize();
        match self {
            Value::I32(x) => x.into(),
            Value::I64(x) => x.into(),
            Value::F32(x) => x.to_bits().into(),
            Value::F64(x) => x.to_bits().into(),
            Value::RefNull => Bytes32::default(),
            Value::Ref(_, x) => x,
            Value::RefExtern(_, x) => x,
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
            ValueType::RefNull | ValueType::FuncRef | ValueType::ExternRef => Value::RefNull,
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
