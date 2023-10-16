// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::{
    io,
    io::{BufReader, BufWriter, Read, Write},
    net::TcpStream,
};

use crate::wavmio::Bytes32;

pub const SUCCESS: u8 = 0x0;
pub const FAILURE: u8 = 0x1;
pub const PREIMAGE: u8 = 0x2;
pub const ANOTHER: u8 = 0x3;
pub const READY: u8 = 0x4;

pub fn read_u8<T: Read>(reader: &mut BufReader<T>) -> Result<u8, io::Error> {
    let mut buf = [0; 1];
    reader.read_exact(&mut buf).map(|_| u8::from_be_bytes(buf))
}

pub fn read_u32<T: Read>(reader: &mut BufReader<T>) -> Result<u32, io::Error> {
    let mut buf = [0; 4];
    reader.read_exact(&mut buf).map(|_| u32::from_be_bytes(buf))
}

pub fn read_u64<T: Read>(reader: &mut BufReader<T>) -> Result<u64, io::Error> {
    let mut buf = [0; 8];
    reader.read_exact(&mut buf).map(|_| u64::from_be_bytes(buf))
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

pub fn read_boxed_slice<T: Read>(reader: &mut BufReader<T>) -> Result<Box<[u8]>, io::Error> {
    Ok(Vec::into_boxed_slice(read_bytes(reader)?))
}

pub fn write_u8(writer: &mut BufWriter<TcpStream>, data: u8) -> Result<(), io::Error> {
    let buf = [data; 1];
    writer.write_all(&buf)
}

pub fn write_u64(writer: &mut BufWriter<TcpStream>, data: u64) -> Result<(), io::Error> {
    let buf = data.to_be_bytes();
    writer.write_all(&buf)
}

pub fn write_bytes32(writer: &mut BufWriter<TcpStream>, data: &Bytes32) -> Result<(), io::Error> {
    writer.write_all(data)
}

pub fn write_bytes(writer: &mut BufWriter<TcpStream>, data: &[u8]) -> Result<(), io::Error> {
    write_u64(writer, data.len() as u64)?;
    writer.write_all(data)
}
