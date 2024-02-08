extern crate libc;

use libc::c_int;
use std::ptr;

type HashFunction = extern "C" fn(*mut u8, *const u8, u64);

extern "C" {
    fn hashtree_init(override_: *const HashFunction) -> c_int;
    fn hashtree_hash(output: *mut u8, input: *const u8, count: u64);
}

pub fn init() -> i32 {
    unsafe { hashtree_init(ptr::null()) }
}

pub fn hash(out: &mut [u8], chunks: &[u8], count: usize) {
    unsafe { hashtree_hash(out.as_mut_ptr(), chunks.as_ptr(), count as u64) }
}
