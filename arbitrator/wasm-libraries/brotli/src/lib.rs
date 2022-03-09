use go_abi::*;

extern "C" {
    pub fn BrotliDecoderDecompress(
        encoded_size: usize,
        encoded_buffer: *const u8,
        decoded_size: *mut usize,
        decoded_buffer: *mut u8,
    ) -> u32;

    pub fn BrotliEncoderCompress(
        quality: u32,
        lgwin: u32,
        mode: u32,
        input_size: usize,
        input_buffer: *const u8,
        encoded_size: *mut usize,
        encoded_buffer: *mut u8,
    ) -> u32;
}

const BROTLI_MODE_GENERIC: u32 = 0;
const BROTLI_RES_SUCCESS: u32 = 1;

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbcompress_brotliDecompress(
    sp: GoStack,
) {
    //(inBuf []byte, outBuf []byte) int
    let in_buf_ptr = sp.read_u64(0);
    let in_buf_len = sp.read_u64(1);
    let out_buf_ptr = sp.read_u64(3);
    let out_buf_len = sp.read_u64(4);
    const OUTPUT_ARG: usize = 6;

    let in_slice = read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;
    let res = BrotliDecoderDecompress(
        in_buf_len as usize,
        in_slice.as_ptr(),
        &mut output_len,
        output.as_mut_ptr(),
    );
    if (res != BROTLI_RES_SUCCESS) || (output_len as u64 > out_buf_len) {
        sp.write_u64(OUTPUT_ARG, u64::MAX);
        return;
    }
    write_slice(&output[..output_len], out_buf_ptr);
    sp.write_u64(OUTPUT_ARG, output_len as u64);
    return;
}

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbcompress_brotliCompress(sp: GoStack) {
    //(inBuf []byte, outBuf []byte, level int, windowSize int) int
    let in_buf_ptr = sp.read_u64(0);
    let in_buf_len = sp.read_u64(1);
    let out_buf_ptr = sp.read_u64(3);
    let out_buf_len = sp.read_u64(4);
    let level = sp.read_u64(6) as u32;
    let windowsize = sp.read_u64(7) as u32;
    const OUTPUT_ARG: usize = 8;

    let in_slice = read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;
    let res = BrotliEncoderCompress(
        level,
        windowsize,
        BROTLI_MODE_GENERIC,
        in_buf_len as usize,
        in_slice.as_ptr(),
        &mut output_len,
        output.as_mut_ptr(),
    );
    if (res != BROTLI_RES_SUCCESS) || (output_len as u64 > out_buf_len) {
        sp.write_u64(OUTPUT_ARG, u64::MAX);
        return;
    }
    write_slice(&output[..output_len], out_buf_ptr);
    sp.write_u64(OUTPUT_ARG, output_len as u64);
    return;
}
