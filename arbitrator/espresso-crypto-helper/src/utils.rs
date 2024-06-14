use serde::{
    de::{DeserializeOwned, Deserializer, Error as _},
    ser::Serializer,
    Deserialize, Serialize,
};

pub type Err = u8;

/// Basically copied from sequencer repo. Just removed the error types to avoid introducing other crate
pub trait FromStringOrInteger: Sized {
    type Binary: Serialize + DeserializeOwned;
    type Integer: Serialize + DeserializeOwned;

    fn from_binary(b: Self::Binary) -> Result<Self, Err>;
    fn from_string(s: String) -> Result<Self, Err>;
    fn from_integer(i: Self::Integer) -> Result<Self, Err>;

    fn to_binary(&self) -> Result<Self::Binary, Err>;
    fn to_string(&self) -> Result<String, Err>;
}

#[macro_export]
macro_rules! impl_serde_from_string_or_integer {
    ($t:ty) => {
        impl serde::Serialize for $t {
            fn serialize<S: serde::ser::Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
                $crate::utils::string_or_integer::serialize(self, s)
            }
        }

        impl<'de> serde::Deserialize<'de> for $t {
            fn deserialize<D: serde::de::Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
                $crate::utils::string_or_integer::deserialize(d)
            }
        }
    };
}

pub use crate::impl_serde_from_string_or_integer;

pub mod string_or_integer {

    use super::*;

    #[derive(Debug, Deserialize)]
    #[serde(untagged)]
    enum StringOrInteger<I> {
        String(String),
        Integer(I),
    }

    pub fn serialize<T: FromStringOrInteger, S: Serializer>(
        t: &T,
        s: S,
    ) -> Result<S::Ok, S::Error> {
        if s.is_human_readable() {
            t.to_string().unwrap().serialize(s)
        } else {
            t.to_binary().unwrap().serialize(s)
        }
    }

    pub fn deserialize<'a, T: FromStringOrInteger, D: Deserializer<'a>>(
        d: D,
    ) -> Result<T, D::Error> {
        if d.is_human_readable() {
            match StringOrInteger::deserialize(d)? {
                StringOrInteger::String(s) => T::from_string(s).map_err(D::Error::custom),
                StringOrInteger::Integer(i) => T::from_integer(i).map_err(D::Error::custom),
            }
        } else {
            T::from_binary(T::Binary::deserialize(d)?).map_err(D::Error::custom)
        }
    }
}
