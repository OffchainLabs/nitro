use std::io;

mod primitives;
mod receiver;
mod sender;
#[cfg(test)]
mod tests;
mod markers;

pub use receiver::*;
pub use sender::*;

pub type IOResult<T> = Result<T, io::Error>;
