use arbutil::Bytes32;
use std::io;
use std::io::{Read, Write};

const SUCCESS: u8 = 0x0;
const FAILURE: u8 = 0x1;
const PREIMAGE: u8 = 0x2;
const ANOTHER: u8 = 0x3;
const READY: u8 = 0x4;

type IOResult<T> = Result<T, io::Error>;

fn read_u8(reader: &mut impl Read) -> IOResult<u8> {
    let mut buf = [0; 1];
    reader.read_exact(&mut buf).map(|_| u8::from_be_bytes(buf))
}

fn write_u8(writer: &mut impl Write, data: u8) -> IOResult<()> {
    let buf = [data; 1];
    writer.write_all(&buf)
}

fn read_u32(reader: &mut impl Read) -> IOResult<u32> {
    let mut buf = [0; 4];
    reader.read_exact(&mut buf).map(|_| u32::from_be_bytes(buf))
}

fn write_u32(writer: &mut impl Write, data: u32) -> IOResult<()> {
    let buf = data.to_be_bytes();
    writer.write_all(&buf)
}

fn read_u64(reader: &mut impl Read) -> IOResult<u64> {
    let mut buf = [0; 8];
    reader.read_exact(&mut buf).map(|_| u64::from_be_bytes(buf))
}

fn write_u64(writer: &mut impl Write, data: u64) -> IOResult<()> {
    let buf = data.to_be_bytes();
    writer.write_all(&buf)
}

fn read_bytes32(reader: &mut impl Read) -> IOResult<Bytes32> {
    let mut buf = [0u8; 32];
    reader.read_exact(&mut buf).map(|_| buf.into())
}

fn write_bytes32(writer: &mut impl Write, data: &Bytes32) -> IOResult<()> {
    writer.write_all(data.as_slice())
}

fn read_bytes(reader: &mut impl Read) -> IOResult<Vec<u8>> {
    let size = read_u64(reader)?;
    let mut buf = vec![0; size as usize];
    reader.read_exact(&mut buf)?;
    Ok(buf)
}

fn write_bytes(writer: &mut impl Write, data: &[u8]) -> IOResult<()> {
    write_u64(writer, data.len() as u64)?;
    writer.write_all(data)
}

fn read_boxed_slice(reader: &mut impl Read) -> IOResult<Box<[u8]>> {
    Ok(Vec::into_boxed_slice(read_bytes(reader)?))
}
