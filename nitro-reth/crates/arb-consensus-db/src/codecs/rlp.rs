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

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use alloy_rlp::{Decodable, Encodable};

    use super::*;

    fn enc<T: Encodable>(v: &T) -> Vec<u8> {
        let mut b = Vec::new();
        v.encode(&mut b);
        b
    }

    #[test]
    fn nil_list_none_is_empty_list_code() {
        let none: NilList<B256> = NilList(None);
        assert_eq!(enc(&none), [EMPTY_LIST_CODE]);
        assert_eq!(NilList::<B256>::decode(&mut &enc(&none)[..]).unwrap(), none);
    }

    #[test]
    fn nil_list_some_roundtrips() {
        let some = NilList(Some(B256::repeat_byte(0xAB)));
        let bytes = enc(&some);
        // 32-byte string: 0xa0 header followed by the raw hash.
        assert_eq!(bytes[0], 0xa0);
        assert_eq!(&bytes[1..], B256::repeat_byte(0xAB).as_slice());
        assert_eq!(NilList::<B256>::decode(&mut &bytes[..]).unwrap(), some);
    }

    #[test]
    fn nil_string_none_is_empty_string_code() {
        let none: NilString<B256> = NilString(None);
        assert_eq!(enc(&none), [EMPTY_STRING_CODE]);
        assert_eq!(
            NilString::<B256>::decode(&mut &enc(&none)[..]).unwrap(),
            none
        );
    }

    #[test]
    fn nil_string_some_roundtrips() {
        let some = NilString(Some(B256::repeat_byte(0x11)));
        assert_eq!(
            NilString::<B256>::decode(&mut &enc(&some)[..]).unwrap(),
            some
        );
    }
}
