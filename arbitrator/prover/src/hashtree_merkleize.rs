use crate::hashtree::hash;
use lazy_static::lazy_static;
use rayon::prelude::*;

const BYTES_PER_CHUNK: usize = 32;

lazy_static! {
    pub static ref ZERO_HASH_ARRAY: [[u8; 32]; 28] = {
        let mut arr = [[0u8; 32]; 28];
        for i in 1..arr.len() {
            {
                let mut temp_arr = [0u8; 32];
                hash_2_chunks(&mut temp_arr[..], &arr[i - 1], &arr[i - 1]);
                arr[i] = temp_arr;
            }
        }
        arr
    };
}

fn compute_hashtree_size(mut chunk_count: usize, mut depth: usize) -> usize {
    let mut ret: usize = 0;
    if depth == 0 {
        return BYTES_PER_CHUNK;
    }
    while depth > 0 {
        if chunk_count & 1 != 0 {
            chunk_count += 1;
        }
        ret += chunk_count >> 1;
        chunk_count >>= 1;
        depth -= 1;
    }
    ret * BYTES_PER_CHUNK
}

fn hash_2_chunks(out: &mut [u8], first: &[u8], second: &[u8]) {
    let mut chunk = Vec::with_capacity(2 * BYTES_PER_CHUNK);
    chunk.extend_from_slice(first);
    chunk.extend_from_slice(second);
    hash(out, chunk.as_slice(), 1);
}

fn sparse_hashtree_in_place(hash_tree: &mut [u8], chunks: &[u8], byte_length: usize, depth: usize) {
    if depth == 0 {
        let end = byte_length.min(chunks.len()).min(hash_tree.len());
        hash_tree[..end].copy_from_slice(&chunks[..end])
    }

    let mut count = (chunks.len() + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK;
    hash(hash_tree, chunks, count / 2);
    let (mut old_layer, mut hash_tree) = hash_tree.split_at_mut(chunks.len() / 2);
    for height in 1..depth {
        count = (old_layer.len() + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK;
        if count > 1 {
            hash(hash_tree, old_layer, count >> 1);
        }
        if count & 1 == 0 {
            (old_layer, hash_tree) = hash_tree.split_at_mut(count * BYTES_PER_CHUNK / 2);
        } else {
            {
                let (_, last) = hash_tree.split_at_mut(count * BYTES_PER_CHUNK / 2);
                hash_2_chunks(
                    last,
                    &old_layer[old_layer.len() - BYTES_PER_CHUNK..],
                    &ZERO_HASH_ARRAY[height],
                );
            }
            (old_layer, hash_tree) = hash_tree.split_at_mut((count + 1) * BYTES_PER_CHUNK / 2);
        }
    }
}

// sparse_hashtree takes a byte slice and merkleizes it as a list of arrays of 32 bytes, with the
// passed limit. It returns a vector of the full hashtree (except the leaves that are constantly
// kept in the passed argument).
pub fn sparse_hashtree(chunks: &[u8], limit: usize) -> Vec<u8> {
    let chunk_count = (chunks.len() + BYTES_PER_CHUNK - 1) / BYTES_PER_CHUNK;
    let depth = if limit == 0 {
        helpers::log2ceil(chunk_count)
    } else {
        helpers::log2ceil(limit)
    };
    if chunk_count == 0 {
        return ZERO_HASH_ARRAY[depth].to_vec();
    }

    let mut ret = vec![0u8; compute_hashtree_size(chunk_count, depth)];
    sparse_hashtree_in_place(ret.as_mut_slice(), chunks, chunks.len(), depth);
    ret
}

mod helpers {
    pub fn log2ceil(n: usize) -> usize {
        if n == 0 {
            0
        } else {
            let bits = std::mem::size_of::<usize>() * 8;
            let leading_zeros = n.leading_zeros() as usize;
            bits - leading_zeros - 1 + if n.count_ones() > 1 { 1 } else { 0 }
        }
    }
}
