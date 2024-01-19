// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{RustBytes, RustSlice};
use arbutil::evm::{api::EvmApiMethod, api::EvmApiStatus, req::RequestHandler};

#[repr(C)]
pub struct NativeRequestHandler {
    pub handle_request: unsafe extern "C" fn(
        id: usize,
        req_type: u32,
        data: *mut RustSlice,
        gas_cost: *mut u64,
        output: *mut RustBytes,
    ) -> EvmApiStatus, // value
    pub id: usize,
}

macro_rules! ptr {
    ($expr:expr) => {
        &mut $expr as *mut _
    };
}
macro_rules! call {
    ($self:expr, $func:ident $(,$message:expr)*) => {
        unsafe { ($self.$func)($self.id $(,$message)*) }
    };
}

impl RequestHandler for NativeRequestHandler {
    fn handle_request(&mut self, req_type: EvmApiMethod, req_data: &[u8]) -> (Vec<u8>, u64) {
        let mut output = RustBytes::new(vec![]);
        let mut cost = 0;
        call!(
            self,
            handle_request,
            req_type as u32 + 0x10000000,
            ptr!(RustSlice::new(req_data)),
            ptr!(cost),
            ptr!(output)
        );
        unsafe { (output.into_vec(), cost) }
    }
}
