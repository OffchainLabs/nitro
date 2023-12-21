// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn program_continue(response: u32, ignored: u32) -> u32;
    fn program_call_main(module: u32, args_len: usize) -> u32;
}

#[no_mangle]
pub unsafe extern "C" fn programs__startProgram(
    module: u32,
    args_len: u32,
) -> u32 {
    // call the program
    program_call_main(module, args_len as usize)
}

#[no_mangle]
pub unsafe extern "C" fn programs__sendResponse(
    req_id: u32,
) -> u32 {
    // call the program
    program_continue(req_id, 0)
}
