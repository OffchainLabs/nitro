extern "C" {
    pub fn wavm_read_pre_image(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
}

#[repr(C, align(32))]
struct Bytes32([u8; 32]);

fn main() {
    println!("hello!");
    unsafe {
        let mut bytebuffer = Bytes32([0x0; 32]);
        // in delayed inbox - we're skipping the "kind" byte
        println!("delayed inbox message 0");
        let len = wavm_read_delayed_inbox_message(0, bytebuffer.0.as_mut_ptr(), 160);
        assert_eq!(len, 2);
        assert_eq!(bytebuffer.0[1], 0xaa);
        println!("delayed inbox message 1");
        let len = wavm_read_delayed_inbox_message(1, bytebuffer.0.as_mut_ptr(), 160);
        assert_eq!(len, 32);
        for j in 1..31 {
            assert_eq!(bytebuffer.0[j], (j as u8));
        }
        println!("inbox message 0");
        let len = wavm_read_inbox_message(0, bytebuffer.0.as_mut_ptr(), 40);
        assert_eq!(len, 1);
        assert_eq!(bytebuffer.0[0], 0xaa);
        println!("inbox message 1");
        let len = wavm_read_inbox_message(1, bytebuffer.0.as_mut_ptr(), 40);
        assert_eq!(len, 32);
        for j in 0..32 {
            assert_eq!(bytebuffer.0[j], (j as u8) + 1);
        }
    }
    println!("Done!");
}