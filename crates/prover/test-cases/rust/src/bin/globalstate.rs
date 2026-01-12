extern "C" {
    pub fn wavm_get_globalstate_bytes32(idx: u32, ptr: *mut u8);
    pub fn wavm_set_globalstate_bytes32(idx: u32, ptr: *const u8);
    pub fn wavm_get_globalstate_u64(idx: u32) -> u64;
    pub fn wavm_set_globalstate_u64(idx: u32, val: u64);
}

const BYTES32_VALS_NUM: usize = 2;
const U64_VALS_NUM: usize = 2;

#[repr(C, align(32))]
struct Bytes32([u8; 32]);

fn main() {
    println!("hello!");
    unsafe {
        for i in 0..U64_VALS_NUM {
            println!("set_u64 nr. {}", i);
            wavm_set_globalstate_u64(i as u32, (i as u64) * 0x100 + 1);
        }
        for i in 0..BYTES32_VALS_NUM {
            let mut bytebuffer = Bytes32([0x0; 32]);
            for j in 0..32 {
                bytebuffer.0[j] = (j + i*0x10) as u8;
            }
            println!("set_bytes32 nr. {}", i);
            wavm_set_globalstate_bytes32(i as u32, bytebuffer.0.as_ptr());
        }
        for i in 0..U64_VALS_NUM {
            println!("get_u64 nr. {}", i);
            let val: u64;
            val = wavm_get_globalstate_u64(i as u32);
            let exp = (i as u64) * 0x100 + 1;
            if val != exp {
                panic!("globalstate u64 val {} expected {} got {}", i, exp, val)
            }
        }
        for i in 0..BYTES32_VALS_NUM {
            let mut bytebuffer = Bytes32([0xff; 32]);
            println!("get_bytes32 nr. {}", i);
            wavm_get_globalstate_bytes32(i as u32, bytebuffer.0.as_mut_ptr());
            let localarray: [u8; 32] = bytebuffer.0;
            for j in 0..32 {
                let foundval = localarray[j];
                println!("byte {} found {}", j, foundval);
                if foundval != (j + i*0x10) as u8 {
                    panic!("globalstate bytes32 idx {} byte {} expected {} got{}",
                            i, j, j + i*0x10, foundval)
                }
            }
        }
    }
    println!("Done!");
}