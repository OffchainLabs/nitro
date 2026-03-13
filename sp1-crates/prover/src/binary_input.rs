use bytes::{Bytes, BytesMut};

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
