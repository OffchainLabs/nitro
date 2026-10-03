// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

/// Without the `libc` feature, `CBytes` is just `Vec<u8>`.
/// With `libc`, it is a manually-allocated buffer using `libc::malloc/free` — necessary for FFI
/// where Go allocates memory that Rust must free.
#[cfg(not(feature = "libc"))]
pub type CBytes = Vec<u8>;

#[cfg(feature = "libc")]
pub use libc_cbytes::{CBytes, CBytesIntoIter};

#[cfg(feature = "libc")]
mod libc_cbytes {
    use std::{borrow::Borrow, fmt, ops::Deref};

    pub struct CBytes {
        ptr: *mut u8,
        len: usize,
    }

    impl CBytes {
        pub fn new() -> Self {
            Self::default()
        }

        pub fn as_slice(&self) -> &[u8] {
            if self.ptr.is_null() {
                return &[];
            }
            unsafe { std::slice::from_raw_parts(self.ptr, self.len) }
        }

        /// Takes ownership of a pointer allocated with `libc::malloc`.
        /// The caller must ensure `ptr` was allocated with `libc::malloc` and won't be
        /// freed elsewhere.
        pub unsafe fn from_raw_parts(ptr: *mut u8, len: usize) -> Self {
            Self { ptr, len }
        }
    }

    impl Default for CBytes {
        fn default() -> Self {
            Self {
                ptr: std::ptr::null_mut(),
                len: 0,
            }
        }
    }

    impl fmt::Debug for CBytes {
        fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            write!(f, "{:?}", self.as_slice())
        }
    }

    impl From<&[u8]> for CBytes {
        fn from(slice: &[u8]) -> Self {
            if slice.is_empty() {
                return Self::default();
            }
            unsafe {
                let ptr = libc::malloc(slice.len()) as *mut u8;
                if ptr.is_null() {
                    panic!("Failed to allocate memory instantiating CBytes");
                }
                std::ptr::copy_nonoverlapping(slice.as_ptr(), ptr, slice.len());
                Self {
                    ptr,
                    len: slice.len(),
                }
            }
        }
    }

    // This type is basically a Box<[u8]> (which is Send + Sync) with libc as an allocator.
    // Any data races between threads are prevented by Rust borrowing rules,
    // and the data isn't thread-local so there's no concern moving it between threads.
    unsafe impl Send for CBytes {}
    unsafe impl Sync for CBytes {}

    impl Drop for CBytes {
        fn drop(&mut self) {
            if !self.ptr.is_null() {
                unsafe {
                    libc::free(self.ptr as *mut _);
                }
            }
        }
    }

    impl Clone for CBytes {
        fn clone(&self) -> Self {
            self.as_slice().into()
        }
    }

    impl Deref for CBytes {
        type Target = [u8];

        fn deref(&self) -> &[u8] {
            self.as_slice()
        }
    }

    impl AsRef<[u8]> for CBytes {
        fn as_ref(&self) -> &[u8] {
            self.as_slice()
        }
    }

    impl Borrow<[u8]> for CBytes {
        fn borrow(&self) -> &[u8] {
            self.as_slice()
        }
    }

    #[derive(Clone)]
    pub struct CBytesIntoIter(CBytes, usize);

    impl Iterator for CBytesIntoIter {
        type Item = u8;

        fn next(&mut self) -> Option<u8> {
            if self.1 >= self.0.len {
                return None;
            }
            let byte = self.0[self.1];
            self.1 += 1;
            Some(byte)
        }

        fn size_hint(&self) -> (usize, Option<usize>) {
            let len = self.0.len - self.1;
            (len, Some(len))
        }
    }

    impl IntoIterator for CBytes {
        type Item = u8;
        type IntoIter = CBytesIntoIter;

        fn into_iter(self) -> CBytesIntoIter {
            CBytesIntoIter(self, 0)
        }
    }
}
