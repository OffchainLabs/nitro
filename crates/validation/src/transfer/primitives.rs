// Copyright 2026-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::transfer::IOResult;
use arbutil::Bytes32;
use std::io::{Read, Write};

pub fn read_u8(reader: &mut impl Read) -> IOResult<u8> {
    let mut buf = [0; 1];
    reader.read_exact(&mut buf).map(|_| u8::from_be_bytes(buf))
}

pub fn write_u8(writer: &mut impl Write, data: u8) -> IOResult<()> {
    let buf = [data; 1];
    writer.write_all(&buf)
}

pub fn read_u32(reader: &mut impl Read) -> IOResult<u32> {
    let mut buf = [0; 4];
    reader.read_exact(&mut buf).map(|_| u32::from_be_bytes(buf))
}

pub fn write_u32(writer: &mut impl Write, data: u32) -> IOResult<()> {
    let buf = data.to_be_bytes();
    writer.write_all(&buf)
}

pub fn read_u64(reader: &mut impl Read) -> IOResult<u64> {
    let mut buf = [0; 8];
    reader.read_exact(&mut buf).map(|_| u64::from_be_bytes(buf))
}

pub fn write_u64(writer: &mut impl Write, data: u64) -> IOResult<()> {
    let buf = data.to_be_bytes();
    writer.write_all(&buf)
}

pub fn read_bytes32(reader: &mut impl Read) -> IOResult<Bytes32> {
    let mut buf = [0u8; 32];
    reader.read_exact(&mut buf).map(|_| buf.into())
}

pub fn write_bytes32(writer: &mut impl Write, data: &Bytes32) -> IOResult<()> {
    writer.write_all(data.as_slice())
}

pub fn read_bytes(reader: &mut impl Read) -> IOResult<Vec<u8>> {
    let size = read_u64(reader)?;
    let mut buf = vec![0; size as usize];
    reader.read_exact(&mut buf)?;
    Ok(buf)
}

pub fn write_bytes(writer: &mut impl Write, data: &[u8]) -> IOResult<()> {
    write_u64(writer, data.len() as u64)?;
    writer.write_all(data)
}
