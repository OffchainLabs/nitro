// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    machine::hash_stack,
    value::{ProgramCounter, Value},
};
use arbutil::{Bytes32, Color};
use digest::Digest;
use sha3::Keccak256;
use std::{
    fmt::{self, Display},
    ops::{Deref, DerefMut},
};

#[derive(Clone, Debug, Default)]
pub(crate) struct ErrorGuardStack {
    pub guards: Vec<ErrorGuard>,
    pub enabled: bool,
}

impl Deref for ErrorGuardStack {
    type Target = Vec<ErrorGuard>;

    fn deref(&self) -> &Self::Target {
        &self.guards
    }
}

impl DerefMut for ErrorGuardStack {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.guards
    }
}

#[derive(Clone, Debug)]
pub struct ErrorGuard {
    pub frame_stack: usize,
    pub value_stack: usize,
    pub inter_stack: usize,
    pub on_error: ProgramCounter,
}

impl Display for ErrorGuard {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{}{} {} {} {} {}{}",
            "ErrorGuard(".grey(),
            self.frame_stack.mint(),
            self.value_stack.mint(),
            self.inter_stack.mint(),
            "→".grey(),
            self.on_error,
            ")".grey(),
        )
    }
}

#[derive(Clone, Debug)]
pub(crate) struct ErrorGuardProof {
    frame_stack: Bytes32,
    value_stack: Bytes32,
    inter_stack: Bytes32,
    on_error: ProgramCounter,
}

impl ErrorGuardProof {
    const STACK_PREFIX: &'static str = "Guard stack:";
    const GUARD_PREFIX: &'static str = "Error guard:";

    pub fn new(
        frame_stack: Bytes32,
        value_stack: Bytes32,
        inter_stack: Bytes32,
        on_error: ProgramCounter,
    ) -> Self {
        Self {
            frame_stack,
            value_stack,
            inter_stack,
            on_error,
        }
    }

    pub fn serialize_for_proof(&self) -> Vec<u8> {
        let mut data = self.frame_stack.to_vec();
        data.extend(self.value_stack.0);
        data.extend(self.inter_stack.0);
        data.extend(Value::from(self.on_error).serialize_for_proof());
        data
    }

    fn hash(&self) -> Bytes32 {
        Keccak256::new()
            .chain(Self::GUARD_PREFIX)
            .chain(self.frame_stack)
            .chain(self.value_stack)
            .chain(self.inter_stack)
            .chain(Value::InternalRef(self.on_error).hash())
            .finalize()
            .into()
    }

    pub fn hash_guards(guards: &[Self]) -> Bytes32 {
        hash_stack(guards.iter().map(|g| g.hash()), Self::STACK_PREFIX)
    }
}
