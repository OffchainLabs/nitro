use crate::{BatchInfo, GoGlobalState, PreimageMap, ValidationInput};
use arbutil::Bytes32;
use io::ErrorKind::InvalidData;
use std::io;
use std::io::{Read, Write};

mod primitives;
mod receiver;

pub use receiver::receive_validation_input;

const SUCCESS: u8 = 0x0;
const FAILURE: u8 = 0x1;
const PREIMAGE: u8 = 0x2;
const ANOTHER: u8 = 0x3;
const READY: u8 = 0x4;

pub type IOResult<T> = Result<T, io::Error>;

