use std::io;

mod primitives;
mod receiver;
mod sender;

pub use receiver::receive_validation_input;
pub use sender::send_validation_input;

pub type IOResult<T> = Result<T, io::Error>;

const SUCCESS: u8 = 0x0;
const FAILURE: u8 = 0x1;
// const PREIMAGE: u8 = 0x2; // legacy, not used
const ANOTHER: u8 = 0x3;
const READY: u8 = 0x4;
