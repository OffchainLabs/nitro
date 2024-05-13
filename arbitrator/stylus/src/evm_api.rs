// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{GoSliceData, RustSlice};
use arbutil::evm::{
    api::{EvmApiMethod, EVM_API_METHOD_REQ_OFFSET},
    req::RequestHandler,
};

#[repr(C)]
pub struct NativeRequestHandler {
    pub handle_request_fptr: unsafe extern "C" fn(
        id: usize,
        req_type: u32,
        data: *mut RustSlice,
        gas_cost: *mut u64,
        result: *mut GoSliceData,
        raw_data: *mut GoSliceData,
    ),
    pub id: usize,
}

macro_rules! ptr {
    ($expr:expr) => {
        &mut $expr as *mut _
    };
}

impl RequestHandler<GoSliceData> for NativeRequestHandler {
    fn request(
        &mut self,
        req_type: EvmApiMethod,
        req_data: impl AsRef<[u8]>,
    ) -> (Vec<u8>, GoSliceData, u64) {
        let mut result = GoSliceData::null();
        let mut raw_data = GoSliceData::null();
        let mut cost = 0;
        unsafe {
            (self.handle_request_fptr)(
                self.id,
                req_type as u32 + EVM_API_METHOD_REQ_OFFSET,
                ptr!(RustSlice::new(req_data.as_ref())),
                ptr!(cost),
                ptr!(result),
                ptr!(raw_data),
            )
        };
        (result.slice().to_vec(), raw_data, cost)
    }
}
