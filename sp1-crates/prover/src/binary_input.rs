use bytes::{Bytes, BytesMut};
use std::io::Read;
use validation::{UserWasm, ValidationInput};

// SP1 has additional alignment requirements, we have to decompress the data
// into aligned bytes
pub fn decompress_aligned(user_wasm: &UserWasm) -> Bytes {
    // This is less ideal but until one of the following happens, we
    // will have to stick with it:
    // * Allocator allocates aligned memory
    // * Bytes add alignment options
    // * Wasmer's Module does not simply accept `IntoBytes` trait.
    align_bytes(&user_wasm.as_vec())
}

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
