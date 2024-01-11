use hex_literal::hex;

extern "C" {
    pub fn wavm_read_keccak_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_sha2_256_preimage(ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_inbox_message(msg_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_delayed_inbox_message(seq_num: u64, ptr: *mut u8, offset: usize) -> usize;
    pub fn wavm_read_hotshot_commitment(ptr: *mut u8, height: u64);
    pub fn wavm_halt_and_set_finished();
}

#[repr(C, align(32))]
struct Bytes32([u8; 32]);

const INBOX_HEADER_LEN: usize = 40; // also in src/main.rs & contracts's OneStepProverHostIo.sol
const DELAYED_HEADER_LEN: usize = 112; // also in src/main.rs & contracts's OneStepProverHostIo.sol

fn main() {
    println!("hello!");
    unsafe {
        let mut bytebuffer = Bytes32([0x0; 32]);
        // in delayed inbox - we're skipping the "kind" byte
        println!("delayed inbox message 0");
        let len = wavm_read_delayed_inbox_message(0, bytebuffer.0.as_mut_ptr(), DELAYED_HEADER_LEN);
        assert_eq!(len, 2);
        assert_eq!(bytebuffer.0[1], 0xaa);
        println!("delayed inbox message 1");
        let len = wavm_read_delayed_inbox_message(1, bytebuffer.0.as_mut_ptr(), DELAYED_HEADER_LEN);
        assert_eq!(len, 32);
        for j in 1..31 {
            assert_eq!(bytebuffer.0[j], (j as u8));
        }
        println!("inbox message 0");
        let len = wavm_read_inbox_message(0, bytebuffer.0.as_mut_ptr(), INBOX_HEADER_LEN);
        assert_eq!(len, 1);
        assert_eq!(bytebuffer.0[0], 0xaa);
        println!("inbox message 1");
        let len = wavm_read_inbox_message(1, bytebuffer.0.as_mut_ptr(), INBOX_HEADER_LEN);
        assert_eq!(len, 32);
        for j in 0..32 {
            assert_eq!(bytebuffer.0[j], (j as u8) + 1);
        }
        let keccak_hash = hex!("47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad");
        bytebuffer = Bytes32(keccak_hash);
        println!("keccak preimage");
        let expected_preimage = b"hello world";
        let len = wavm_read_keccak_256_preimage(bytebuffer.0.as_mut_ptr(), 0);
        assert_eq!(len, expected_preimage.len());
        assert_eq!(&bytebuffer.0[..len], expected_preimage);
        println!("sha2 preimage");
        let sha2_hash = hex!("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9");
        bytebuffer = Bytes32(sha2_hash);
        let len = wavm_read_sha2_256_preimage(bytebuffer.0.as_mut_ptr(), 0);
        assert_eq!(len, expected_preimage.len());
        assert_eq!(&bytebuffer.0[..len], expected_preimage);
    }
    println!("Done!");
}
