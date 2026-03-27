// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use eyre::ErrReport;
use num_enum::{IntoPrimitive, TryFromPrimitive};
use std::fmt::Display;

#[derive(Debug)]
pub enum UserOutcome {
    Success(Vec<u8>),
    Revert(Vec<u8>),
    Failure(ErrReport),
    OutOfInk,
    OutOfStack,
    /// The Wasmer native coroutine stack overflowed (SIGSEGV caught by signal handler).
    /// Unlike OutOfStack (which is the deterministic DepthChecker limit), this indicates
    /// the physical stack was exhausted and the call should be retried with a larger stack.
    NativeStackOverflow,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, TryFromPrimitive, IntoPrimitive)]
#[repr(u8)]
pub enum UserOutcomeKind {
    Success,
    Revert,
    Failure,
    OutOfInk,
    OutOfStack,
    NativeStackOverflow,
}

impl UserOutcome {
    pub fn into_data(self) -> (UserOutcomeKind, Vec<u8>) {
        let kind = self.kind();
        let data = match self {
            Self::Success(out) => out,
            Self::Revert(out) => out,
            Self::Failure(err) => format!("{err:?}").as_bytes().to_vec(),
            _ => vec![],
        };
        (kind, data)
    }

    pub fn kind(&self) -> UserOutcomeKind {
        self.into()
    }
}

impl From<&UserOutcome> for UserOutcomeKind {
    fn from(value: &UserOutcome) -> Self {
        use UserOutcome::*;
        match value {
            Success(_) => Self::Success,
            Revert(_) => Self::Revert,
            Failure(_) => Self::Failure,
            OutOfInk => Self::OutOfInk,
            OutOfStack => Self::OutOfStack,
            NativeStackOverflow => Self::NativeStackOverflow,
        }
    }
}

impl From<&UserOutcome> for u8 {
    fn from(value: &UserOutcome) -> Self {
        UserOutcomeKind::from(value).into()
    }
}

impl Display for UserOutcome {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        use UserOutcome::*;
        match self {
            Success(data) => write!(f, "success {}", hex::encode(data)),
            Failure(err) => write!(f, "failure {err:?}"),
            OutOfInk => write!(f, "out of ink"),
            OutOfStack => write!(f, "out of stack"),
            NativeStackOverflow => write!(f, "native stack overflow"),
            Revert(data) => {
                let text = String::from_utf8(data.clone()).unwrap_or_else(|_| hex::encode(data));
                write!(f, "revert {text}")
            }
        }
    }
}

impl Display for UserOutcomeKind {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let as_u8 = *self as u8;
        use UserOutcomeKind::*;
        match self {
            Success => write!(f, "success ({as_u8})"),
            Revert => write!(f, "revert ({as_u8})"),
            Failure => write!(f, "failure ({as_u8})"),
            OutOfInk => write!(f, "out of ink ({as_u8})"),
            OutOfStack => write!(f, "out of stack ({as_u8})"),
            NativeStackOverflow => write!(f, "native stack overflow ({as_u8})"),
        }
    }
}
