use alloy_primitives::{Address, B256, U256};
use std::io::{self, Read, Write};

/// Reads a 32-byte hash from a reader.
pub fn hash_from_reader<R: Read>(r: &mut R) -> io::Result<B256> {
    let mut buf = [0u8; 32];
    r.read_exact(&mut buf)?;
    Ok(B256::from(buf))
}

/// Writes a 32-byte hash to a writer.
pub fn hash_to_writer<W: Write>(w: &mut W, hash: &B256) -> io::Result<()> {
    w.write_all(hash.as_slice())
}

/// Reads a U256 from a reader (big-endian).
pub fn uint256_from_reader<R: Read>(r: &mut R) -> io::Result<U256> {
    let hash = hash_from_reader(r)?;
    Ok(U256::from_be_bytes(hash.0))
}

/// Reads a 20-byte address from a reader.
pub fn address_from_reader<R: Read>(r: &mut R) -> io::Result<Address> {
    let mut buf = [0u8; 20];
    r.read_exact(&mut buf)?;
    Ok(Address::from(buf))
}

/// Writes a 20-byte address to a writer.
pub fn address_to_writer<W: Write>(w: &mut W, addr: &Address) -> io::Result<()> {
    w.write_all(addr.as_slice())
}

/// Reads a 32-byte padded address from a reader (last 20 bytes are the address).
pub fn address_from_256_from_reader<R: Read>(r: &mut R) -> io::Result<Address> {
    let hash = hash_from_reader(r)?;
    Ok(Address::from_slice(&hash[12..]))
}

/// Writes a 32-byte padded address to a writer (left-padded with zeros).
pub fn address_to_256_to_writer<W: Write>(w: &mut W, addr: &Address) -> io::Result<()> {
    let mut buf = [0u8; 32];
    buf[12..].copy_from_slice(addr.as_slice());
    w.write_all(&buf)
}

/// Reads a uint64 from a reader (big-endian).
pub fn uint64_from_reader<R: Read>(r: &mut R) -> io::Result<u64> {
    let mut buf = [0u8; 8];
    r.read_exact(&mut buf)?;
    Ok(u64::from_be_bytes(buf))
}

/// Writes a uint64 to a writer (big-endian).
pub fn uint64_to_writer<W: Write>(w: &mut W, val: u64) -> io::Result<()> {
    w.write_all(&val.to_be_bytes())
}

/// Reads a length-prefixed byte string from a reader.
///
/// Caps the declared length at `MAX_BYTESTRING_LEN` to prevent an
/// attacker-controlled prefix from triggering a huge allocation (DoS).
/// Any L2 message exceeds this is well past the protocol's 256KiB
/// segment size, so rejecting matches the spec.
pub fn bytestring_from_reader<R: Read>(r: &mut R) -> io::Result<Vec<u8>> {
    /// 1 MiB — generous upper bound; real L2 segments cap at 256 KiB
    /// (see `MAX_L2_MESSAGE_SIZE`). Keeps headroom for non-segment use
    /// sites without opening a DoS vector.
    const MAX_BYTESTRING_LEN: u64 = 1 << 20;

    let len_u64 = uint64_from_reader(r)?;
    if len_u64 > MAX_BYTESTRING_LEN {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("byte-string length {len_u64} exceeds max {MAX_BYTESTRING_LEN}"),
        ));
    }
    let len = len_u64 as usize;
    let mut buf = vec![0u8; len];
    r.read_exact(&mut buf)?;
    Ok(buf)
}

/// Writes a length-prefixed byte string to a writer.
pub fn bytestring_to_writer<W: Write>(w: &mut W, data: &[u8]) -> io::Result<()> {
    uint64_to_writer(w, data.len() as u64)?;
    w.write_all(data)
}

/// Converts a signed integer to a B256 hash (big-endian).
pub fn int_to_hash(val: i64) -> B256 {
    let mut buf = [0u8; 32];
    if val >= 0 {
        buf[24..].copy_from_slice(&(val as u64).to_be_bytes());
    } else {
        // Two's complement for negative values: fill with 0xFF.
        buf.fill(0xFF);
        buf[24..].copy_from_slice(&(val as u64).to_be_bytes());
    }
    B256::from(buf)
}

/// Converts an unsigned integer to a B256 hash (big-endian).
pub fn uint_to_hash(val: u64) -> B256 {
    let mut buf = [0u8; 32];
    buf[24..].copy_from_slice(&val.to_be_bytes());
    B256::from(buf)
}

/// Converts an address to a B256 hash (left-padded with zeros).
pub fn address_to_hash(addr: &Address) -> B256 {
    let mut buf = [0u8; 32];
    buf[12..].copy_from_slice(addr.as_slice());
    B256::from(buf)
}
