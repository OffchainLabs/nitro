// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::value::{GoValue, InterpValue};

#[derive(Clone, Debug)]
pub struct PendingEvent {
    pub id: InterpValue,
    pub this: InterpValue,
    pub args: Vec<GoValue>,
}

pub(crate) static mut PENDING_EVENT: Option<PendingEvent> = None;

pub(crate) unsafe fn set_event(id: InterpValue, this: InterpValue, args: Vec<GoValue>) {
    PENDING_EVENT = Some(PendingEvent { id, this, args });
}

#[no_mangle]
pub unsafe extern "C" fn go_stub_stylus__get_bytes32(key: usize, value: usize) {
    //PENDING_EVENT = Some(*event)
}

#[no_mangle]
pub unsafe extern "C" fn go_stub_stylus__set_bytes32(key: usize, value: usize) {
    //PENDING_EVENT = Some(*event)
}
