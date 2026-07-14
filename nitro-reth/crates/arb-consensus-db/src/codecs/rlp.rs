use alloy_rlp::{Decodable, EMPTY_LIST_CODE, EMPTY_STRING_CODE, Encodable, bytes::BufMut};

/// Optional field, which encodes `None` as empty list (`0xC0`).
///
/// `T` must never be encodable to `0xC0` since it will resolve to `None`.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct NilList<T>(pub Option<T>);

impl<T: Encodable> Encodable for NilList<T> {
    fn encode(&self, out: &mut dyn BufMut) {
        match &self.0 {
            None => out.put_u8(EMPTY_LIST_CODE), // 0xC0
            Some(v) => v.encode(out),
        }
    }

    fn length(&self) -> usize {
        match &self.0 {
            None => 1,
            Some(v) => v.length(),
        }
    }
}

impl<T: Decodable> Decodable for NilList<T> {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        if buf.first() == Some(&EMPTY_LIST_CODE) {
            *buf = &buf[1..];
            return Ok(Self(None));
        }
        Ok(Self(Some(T::decode(buf)?)))
    }
}

/// Optional field, which encodes `None` as empty string (`0x80`).
///
/// `T` must never be encodable to `0x80` since it will resolve to `None`.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct NilString<T>(pub Option<T>);

impl<T: Encodable> Encodable for NilString<T> {
    fn encode(&self, out: &mut dyn BufMut) {
        match &self.0 {
            None => out.put_u8(EMPTY_STRING_CODE), // 0x80
            Some(v) => v.encode(out),
        }
    }

    fn length(&self) -> usize {
        match &self.0 {
            None => 1,
            Some(v) => v.length(),
        }
    }
}

impl<T: Decodable> Decodable for NilString<T> {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        if buf.first() == Some(&EMPTY_STRING_CODE) {
            *buf = &buf[1..];
            return Ok(Self(None));
        }
        Ok(Self(Some(T::decode(buf)?)))
    }
}
