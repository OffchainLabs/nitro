use digest::Digest;
use sha3::Keccak256;

use crate::utils::Bytes32;

#[derive(Clone, Copy, PartialEq, Eq, Debug)]
pub enum ValueType {
    I32,
    I64,
    F32,
    F64,
    RefNull,
    FuncRef,
    ExternRef,
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
    Ref(u32),
    RefExtern(u32),
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
            Value::Ref(_) => ValueType::FuncRef,
            Value::RefExtern(_) => ValueType::ExternRef,
        }
    }

    pub fn contents(mut self) -> u64 {
        self.canonicalize();
        match self {
            Value::I32(x) => x.into(),
            Value::I64(x) => x,
            Value::F32(x) => x.to_bits().into(),
            Value::F64(x) => x.to_bits(),
            Value::RefNull => 0,
            Value::Ref(x) | Value::RefExtern(x) => x.into(),
        }
    }

    pub fn serialize(self) -> [u8; 9] {
        let mut ret = [0u8; 9];
        ret[0] = self.ty().serialize();
        ret[1..].copy_from_slice(&self.contents().to_be_bytes());
        ret
    }

    pub fn hash(self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update(b"Value:");
        h.update(&[self.ty() as u8]);
		h.update(&self.contents().to_be_bytes());
		h.finalize().into()
    }

    pub fn default_of_type(ty: ValueType) -> Value {
        match ty {
            ValueType::I32 => Value::I32(0),
            ValueType::I64 => Value::I64(0),
            ValueType::F32 => Value::F32(0.),
            ValueType::F64 => Value::F64(0.),
            ValueType::RefNull | ValueType::FuncRef | ValueType::ExternRef => Value::RefNull,
        }
    }
}

impl PartialEq for Value {
    fn eq(&self, other: &Self) -> bool {
        self.ty() == other.ty() && self.contents() == other.contents()
    }
}

impl Eq for Value {}
