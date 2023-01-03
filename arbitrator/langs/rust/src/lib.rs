// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#[link(wasm_import_module = "forward")]
extern "C" {
    pub fn read_args(data: *mut u8);
    pub fn return_data(data: *const u8, len: usize);
}

pub fn args(len: usize) -> Vec<u8> {
    let mut input = Vec::with_capacity(len);
    unsafe {
        read_args(input.as_mut_ptr());
        input.set_len(len);
    }
    input
}

pub fn output(data: Vec<u8>) {
    unsafe {
        let len = data.len();
        let out = data.as_ptr();
        std::mem::forget(data); // leak the data
        return_data(out, len);
    }
}

#[macro_export]
macro_rules! arbitrum_main {
    ($name:expr) => {
        #[no_mangle]
        pub extern "C" fn arbitrum_main(len: usize) -> usize {
            let input = arbitrum::args(len);
            let (data, status) = match $name(input) {
                Ok(data) => (data, 0),
                Err(data) => (data, 1),
            };
            arbitrum::output(data);
            status
        }
    };
}
