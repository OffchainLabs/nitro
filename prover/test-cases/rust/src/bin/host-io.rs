extern "C" {
    pub fn wavm_get_globalstate_bytes32(idx: u32, ptr: *mut u8);
    pub fn wavm_set_globalstate_bytes32(idx: u32, ptr: *const u8);
    pub fn wavm_get_globalstate_u64(idx: u32) -> u64;
    pub fn wavm_set_globalstate_u64(idx: u32, val: u64);
    pub fn wavm_read_pre_image(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
}

const BYTES32_VALS_NUM: usize = 1;
const U64_VALS_NUM: usize = 2;

fn main() {
    for i in 0..BYTES32_VALS_NUM {
        let mut bytebuffer = [0x0 as u8; 32];
        for j in 0..32 {
            bytebuffer[j] = (j + i*0x10) as u8;
        }
        unsafe {
            wavm_set_globalstate_bytes32(i as u32, bytebuffer.as_ptr());
        }
    }
    for i in 0..U64_VALS_NUM {
        unsafe {
            wavm_set_globalstate_u64(i as u32, (i as u64) * 0x100);
        }
    }
    for i in 0..BYTES32_VALS_NUM {
        let mut bytebuffer = [0xff as u8; 32];
        unsafe {
            wavm_get_globalstate_bytes32(i as u32, bytebuffer.as_mut_ptr());
        }
        for j in 0..32 {
            if bytebuffer[j] != (j + i*0x10) as u8 {
                panic!("globalstate bytes32 idx {} byte {} expected {} got{}",
                        i, j, j + i*0x10, bytebuffer[j])
            }
        }
    }
    for i in 0..U64_VALS_NUM {
        let val: u64;
        unsafe {
            val = wavm_get_globalstate_u64(i as u32);
        }
        let exp = (i as u64) * 0x100;
        if val != exp {
            panic!("globalstate u64 val {} expected {} got {}", i, exp, val)
        }
    }

}