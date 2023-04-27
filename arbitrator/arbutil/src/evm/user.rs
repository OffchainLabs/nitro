// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use eyre::ErrReport;
use std::fmt::Display;

#[derive(Debug)]
pub enum UserOutcome {
    Success(Vec<u8>),
    Revert(Vec<u8>),
    Failure(ErrReport),
    OutOfInk,
    OutOfStack,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(u8)]
pub enum UserOutcomeKind {
    Success,
    Revert,
    Failure,
    OutOfInk,
    OutOfStack,
}

impl UserOutcome {
    pub fn into_data(self) -> (UserOutcomeKind, Vec<u8>) {
        let kind = (&self).into();
        let data = match self {
            Self::Success(out) => out,
            Self::Revert(out) => out,
            Self::Failure(err) => format!("{err:?}").as_bytes().to_vec(),
            _ => vec![],
        };
        (kind, data)
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
        }
    }
}

impl From<&UserOutcome> for u8 {
    fn from(value: &UserOutcome) -> Self {
        UserOutcomeKind::from(value).into()
    }
}

impl From<UserOutcomeKind> for u8 {
    fn from(value: UserOutcomeKind) -> Self {
        value as u8
    }
}

impl Display for UserOutcome {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        use UserOutcome::*;
        match self {
            Success(data) => write!(f, "success {}", hex::encode(data)),
            Failure(err) => write!(f, "failure {:?}", err),
            OutOfInk => write!(f, "out of ink"),
            OutOfStack => write!(f, "out of stack"),
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
        }
    }
}
