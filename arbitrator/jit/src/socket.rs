// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::{
    io,
    io::{BufReader, Read, Write},
    net::TcpStream,
};

use crate::wavmio::Bytes32;

pub const EXIT_SUCCESS: u8 = 0x0;
pub const EXIT_FAILED: u8 = 0x1;
pub const REQUEST_PREIMAGE: u8 = 0x02;
pub const ANOTHER: u8 = 0x03;
pub const READY: u8 = 0x04;

pub fn read_u8<T: Read>(reader: &mut BufReader<T>) -> Result<u8, io::Error> {
    let mut buf = [0; 1];
    reader.read_exact(&mut buf).map(|_| u8::from_le_bytes(buf))
}

pub fn read_u64<T: Read>(reader: &mut BufReader<T>) -> Result<u64, io::Error> {
    let mut buf = [0; 8];
    reader.read_exact(&mut buf).map(|_| u64::from_le_bytes(buf))
}

pub fn read_bytes32<T: Read>(reader: &mut BufReader<T>) -> Result<Bytes32, io::Error> {
    let mut buf = Bytes32::default();
    reader.read_exact(&mut buf).map(|_| buf)
}

pub fn read_bytes<T: Read>(reader: &mut BufReader<T>) -> Result<Vec<u8>, io::Error> {
    let size = read_u64(reader)?;
    let mut buf = vec![0; size as usize];
    reader.read_exact(&mut buf)?;
    Ok(buf)
}

#[must_use]
pub fn write_u8(writer: &mut TcpStream, data: u8) -> Result<(), io::Error> {
    let buf = [data, 1];
    writer.write_all(&buf)
}

#[must_use]
pub fn write_u64(writer: &mut TcpStream, data: u64) -> Result<(), io::Error> {
    let buf = data.to_le_bytes();
    writer.write_all(&buf)
}

#[must_use]
pub fn write_bytes32(writer: &mut TcpStream, data: &Bytes32) -> Result<(), io::Error> {
    writer.write_all(data)
}

#[must_use]
pub fn write_bytes(writer: &mut TcpStream, data: &[u8]) -> Result<(), io::Error> {
    write_u64(writer, data.len() as u64)?;
    writer.write_all(data)
}
