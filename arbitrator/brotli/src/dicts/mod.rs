// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    types::BrotliSharedDictionaryType, CustomAllocator, EncoderPreparedDictionary, HeapItem,
};
use core::{ffi::c_int, ptr};
use lazy_static::lazy_static;
use num_enum::{IntoPrimitive, TryFromPrimitive};

extern "C" {
    /// Prepares an LZ77 dictionary for use during compression.
    fn BrotliEncoderPrepareDictionary(
        dict_type: BrotliSharedDictionaryType,
        dict_len: c_int,
        dictionary: *const u8,
        quality: c_int,
        alloc: Option<extern "C" fn(opaque: *const CustomAllocator, size: usize) -> *mut HeapItem>,
        free: Option<extern "C" fn(opaque: *const CustomAllocator, address: *mut HeapItem)>,
        opaque: *mut CustomAllocator,
    ) -> *mut EncoderPreparedDictionary;

    /// Nonzero when valid.
    fn BrotliEncoderGetPreparedDictionarySize(
        dictionary: *const EncoderPreparedDictionary,
    ) -> usize;
}

/// Forces a type to implement [`Sync`].
struct ForceSync<T>(T);

unsafe impl<T> Sync for ForceSync<T> {}

lazy_static! {
    /// Memoizes dictionary preperation.
    static ref STYLUS_PROGRAM_DICT: ForceSync<*const EncoderPreparedDictionary> =
        ForceSync(unsafe {
            let data = Dictionary::StylusProgram.slice().unwrap();
            let dict = BrotliEncoderPrepareDictionary(
                BrotliSharedDictionaryType::Raw,
                data.len() as c_int,
                data.as_ptr(),
                11,
                None,
                None,
                ptr::null_mut(),
            );
            assert!(BrotliEncoderGetPreparedDictionarySize(dict) > 0); // check integrity
            dict as _
        });
}

/// Brotli dictionary selection.
#[derive(Clone, Copy, Debug, PartialEq, IntoPrimitive, TryFromPrimitive)]
#[repr(u32)]
pub enum Dictionary {
    Empty,
    StylusProgram,
}

impl Dictionary {
    /// Gets the raw bytes of the underlying LZ77 dictionary.
    pub fn slice(&self) -> Option<&[u8]> {
        match self {
            Self::StylusProgram => Some(include_bytes!("stylus-program-11.lz")),
            _ => None,
        }
    }

    /// Returns a pointer to a compression-ready instance of the given dictionary.
    pub fn ptr(&self, level: u32) -> Option<*const EncoderPreparedDictionary> {
        match self {
            Self::StylusProgram if level == 11 => Some(STYLUS_PROGRAM_DICT.0),
            _ => None,
        }
    }
}

impl From<Dictionary> for u8 {
    fn from(value: Dictionary) -> Self {
        value as u32 as u8
    }
}

impl TryFrom<u8> for Dictionary {
    type Error = <Dictionary as TryFrom<u32>>::Error;

    fn try_from(value: u8) -> Result<Self, Self::Error> {
        (value as u32).try_into()
    }
}
