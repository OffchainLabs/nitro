use crate::{BatchInfo, GoGlobalState, PreimageMap, ValidationInput};
use arbutil::Bytes32;
use io::ErrorKind::InvalidData;
use std::io;
use std::io::{Read, Write};

const SUCCESS: u8 = 0x0;
const FAILURE: u8 = 0x1;
const PREIMAGE: u8 = 0x2;
const ANOTHER: u8 = 0x3;
const READY: u8 = 0x4;

pub type IOResult<T> = Result<T, io::Error>;

pub fn receive_validation_input(reader: &mut impl Read) -> IOResult<ValidationInput> {
    let start_state = receive_global_state(reader)?;
    let inbox = receive_batches(reader)?;
    let delayed_message = receive_delayed_message(reader)?.unwrap_or_default();
    let preimages = receiver_preimages(reader)?;

    Ok(ValidationInput {
        id: 0,
        has_delayed_msg: delayed_message.data.is_empty(),
        delayed_msg_nr: delayed_message.number,
        preimages: Default::default(),
        batch_info: inbox,
        delayed_msg: delayed_message.data,
        start_state,
        user_wasms: Default::default(),
        debug_chain: false,
        max_user_wasm_size: 0,
    })
}

fn receive_global_state(reader: &mut impl Read) -> IOResult<GoGlobalState> {
    let inbox_position = read_u64(reader)?;
    let position_within_message = read_u64(reader)?;
    let last_block_hash = read_bytes32(reader)?;
    let last_send_root = read_bytes32(reader)?;
    Ok(GoGlobalState {
        block_hash: last_block_hash,
        send_root: last_send_root,
        batch: inbox_position,
        pos_in_batch: position_within_message,
    })
}

fn receive_batches(reader: &mut impl Read) -> IOResult<Vec<BatchInfo>> {
    let mut batches = vec![];
    while read_u8(reader)? == ANOTHER {
        let number = read_u64(reader)?;
        let data = read_bytes(reader)?;
        batches.push(BatchInfo { number, data });
    }
    Ok(batches)
}

fn receive_delayed_message(reader: &mut impl Read) -> IOResult<Option<BatchInfo>> {
    match &receive_batches(reader)?[..] {
        [] => Ok(None),
        [batch_info] => Ok(Some(batch_info.clone())),
        _ => Err(io::Error::new(InvalidData, "multiple delayed batches")),
    }
}

fn receiver_preimages(reader: &mut impl Read) -> IOResult<PreimageMap> {
    // Preimages are not implemented yet.
    Ok(())
}

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
