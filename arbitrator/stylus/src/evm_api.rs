// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::ptr::slice_from_raw_parts;

use crate::RustSlice;
use arbutil::evm::{
    api::{
        EvmApiMethod, EVM_API_METHOD_REQ_OFFSET, {DataReader, EvmApiStatus},
    },
    req::RequestHandler,
};

#[derive(Clone, Copy, Default)]
#[repr(C)]
pub struct GoPinnedData {
    ptr: usize, // not stored as pointer because rust won't let that be Send
    len: usize,
}

#[repr(C)]
pub struct NativeRequestHandler {
    pub handle_request_fptr: unsafe extern "C" fn(
        id: usize,
        req_type: u32,
        data: *mut RustSlice,
        gas_cost: *mut u64,
        result: *mut GoPinnedData,
        raw_data: *mut GoPinnedData,
    ) -> EvmApiStatus, // value
    pub id: usize,
}

macro_rules! ptr {
    ($expr:expr) => {
        &mut $expr as *mut _
    };
}

impl DataReader for GoPinnedData {
    fn get(&self) -> &[u8] {
        if self.len == 0 {
            return &[];
        }
        unsafe { &*slice_from_raw_parts(self.ptr as *const u8, self.len) }
    }
}

impl RequestHandler<GoPinnedData> for NativeRequestHandler {
    fn handle_request(
        &mut self,
        req_type: EvmApiMethod,
        req_data: &[u8],
    ) -> (Vec<u8>, GoPinnedData, u64) {
        let mut result = GoPinnedData::default();
        let mut raw_data = GoPinnedData::default();
        let mut cost = 0;
        let status = unsafe {
            (self.handle_request_fptr)(
                self.id,
                req_type as u32 + EVM_API_METHOD_REQ_OFFSET,
                ptr!(RustSlice::new(req_data)),
                ptr!(cost),
                ptr!(result),
                ptr!(raw_data),
            )
        };
        assert_eq!(status, EvmApiStatus::Success);
        (result.get().to_vec(), raw_data, cost)
    }
}
