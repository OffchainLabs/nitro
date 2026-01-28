use std::io;

mod markers;
mod primitives;
mod receiver;
mod sender;
#[cfg(test)]
mod tests;

pub use receiver::*;
pub use sender::*;

pub type IOResult<T> = Result<T, io::Error>;
