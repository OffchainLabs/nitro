use bytes::{Bytes, BytesMut};
use std::io::Read;
use validation::ValidationInput;

/// Copies `data` into 8-byte-aligned memory and returns it as `Bytes`.
/// SP1's wasmer fork requires aligned memory for `Module::deserialize`.
pub fn align_bytes(data: &[u8]) -> Bytes {
    let mut buffer = BytesMut::zeroed(data.len() + 7);
    let p = buffer.as_ptr() as usize;
    let aligned_p = (p + 7) / 8 * 8;
    let offset = aligned_p - p;
    buffer[offset..offset + data.len()].copy_from_slice(data);
    let bytes = buffer.freeze();
    bytes.slice(offset..offset + data.len())
}

pub fn read_validation_input<R: Read>(mut reader: R) -> Result<ValidationInput, String> {
    let mut s = Vec::new();
    reader
        .read_to_end(&mut s)
        .map_err(|e| format!("IO Error: {e:?}"))?;
    let archived =
        rkyv::access::<validation::ArchivedValidationInput, rkyv::rancor::Error>(&s[..])
            .map_err(|e| format!("rkyv access error: {e:?}"))?;
    rkyv::deserialize::<ValidationInput, rkyv::rancor::Error>(archived)
        .map_err(|e| format!("rkyv deserialize error: {e:?}"))
}
