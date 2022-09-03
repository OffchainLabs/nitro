// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::{
    io,
    io::{BufReader, Read},
};

use crate::wavmio::Bytes32;

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
