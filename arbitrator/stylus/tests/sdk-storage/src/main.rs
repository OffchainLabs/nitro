// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::{Address, Signed, Uint, B256, I32, U16, U256, U64, U8},
    prelude::*,
};

#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

sol_storage! {
    pub struct Contract {
        bool flag;
        address owner;
        address other;
        Struct sub;
        Struct[] structs;
        uint64[] vector;
        uint40[][] nested;
        bytes bytes_full;
        bytes bytes_long;
        string chars;
        Maps maps;
        Arrays arrays;
    }

    #[derive(Erase)]
    pub struct Struct {
        uint16 num;
        int32 other;
        bytes32 word;
    }

    pub struct Maps {
        mapping(uint256 => address) basic;
        mapping(address => bool[]) vects;
        mapping(int32 => address)[] array;
        mapping(bytes1 => mapping(bool => uint256)) nested;
        mapping(string => Struct) structs;
    }

    pub struct Arrays {
        string[4] strings;

        uint8 spacer;
        uint24[5] packed;
        uint8 trail;

        address[2] spill;
        uint8[2][4] matrix;
        int96[4][] vector;
        int96[][4] vectors;
        Struct[3] structs;
    }
}

#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let contract = unsafe { Contract::new(U256::ZERO, 0) };
    let selector = u32::from_be_bytes(input[0..4].try_into().unwrap());
    match selector {
        0xf809f205 => populate(contract),
        0xa7f43779 => remove(contract),
        _ => panic!("unknown method"),
    }
    Ok(vec![])
}

fn populate(mut contract: Contract) {
    // test primitives
    let owner = Address::with_last_byte(0x70);
    contract.flag.set(true);
    contract.owner.set(owner);
    assert_eq!(contract.owner.get(), owner);

    let mut setter = contract.other.load_mut();
    setter.set(Address::with_last_byte(0x30));

    // test structs
    contract.sub.num.set(U16::from(32));
    contract.sub.other.set(I32::MAX);
    contract.sub.word.set(U256::from(64).into());
    assert_eq!(contract.sub.num.get(), U16::from(32));
    assert_eq!(contract.sub.other.get(), I32::MAX);
    assert_eq!(contract.sub.word.get(), B256::from(U256::from(64)));

    // test primitive vectors
    let mut vector = contract.vector;
    for i in (0..32).map(U64::from) {
        vector.push(i);
    }
    for i in (0..32).map(U64::from) {
        assert_eq!(vector.get(i), Some(i));
    }
    let value = U64::from(77);
    let mut setter = vector.get_mut(7).unwrap();
    setter.set(value);
    assert_eq!(setter.get(), value);

    // test nested vectors
    let mut nested = contract.nested;
    for w in 0..10 {
        let mut inner = nested.grow();
        for i in 0..w {
            inner.push(Uint::from(i));
        }
        assert_eq!(inner.len(), w);
        assert_eq!(nested.len(), w + 1);
    }
    for w in 0..10 {
        let mut inner = nested.get_mut(w).unwrap();

        for i in 0..w {
            let value = inner.get(i).unwrap() * Uint::from(2);
            let mut setter = inner.get_mut(i).unwrap();
            setter.set(value);
            assert_eq!(inner.get(i), Some(value));
        }
    }

    // test bytes
    let mut bytes_full = contract.bytes_full;
    let mut bytes_long = contract.bytes_long;

    for i in 0..31 {
        bytes_full.push(i);
    }
    for i in 0..80 {
        bytes_long.push(i);
    }
    for i in 0..31 {
        assert_eq!(bytes_full.get(i), Some(i));
    }
    for i in 0..80 {
        let setter = bytes_long.get_mut(i).unwrap();
        assert_eq!(setter.get()[0], i);
    }
    assert_eq!(bytes_full.get(32), None);
    assert_eq!(bytes_long.get(80), None);

    // test strings
    let mut chars = contract.chars;
    assert!(chars.is_empty() && chars.len() == 0);
    assert_eq!(chars.get_string(), "");
    for c in "arbitrum stylus".chars() {
        chars.push(c);
    }
    assert_eq!(chars.get_string(), "arbitrum stylus");

    // test basic maps
    let maps = contract.maps;
    let mut basic = maps.basic;
    for i in (0..16).map(U256::from) {
        basic.insert(i, Address::from_word(B256::from(i)));
    }
    for i in 0..16 {
        assert_eq!(basic.get(U256::from(i)), Address::with_last_byte(i));
    }
    assert_eq!(basic.get(U256::MAX), Address::ZERO);

    // test map of vectors
    let mut vects = maps.vects;
    for a in 0..4 {
        let mut bools = vects.setter(Address::with_last_byte(a));
        for _ in 0..=a {
            bools.push(true)
        }
    }

    // test vector of maps
    let mut array = maps.array;
    for i in 0..4 {
        let mut map = array.grow();
        let value = I32::from_le_bytes::<4>((i as u32).to_le_bytes());
        map.insert(value, Address::with_last_byte(i));
    }

    // test maps of maps
    let mut nested = maps.nested;
    for i in 0..4 {
        let mut inner = nested.setter(U8::from(i).into());
        let mut value = inner.setter(U8::from((i % 2 == 0) as u8));
        value.set(Uint::from(i + 1));
    }

    // test map of structs (TODO: direct assignment)
    let mut structs = maps.structs;
    let mut entry = structs.setter("stylus".to_string());
    entry.num.set(contract.sub.num.get());
    entry.other.set(contract.sub.other.get());
    entry.word.set(contract.sub.word.get());

    // test vec of structs
    let mut structs = contract.structs;
    for _ in 0..4 {
        let mut entry = structs.grow();
        entry.num.set(contract.sub.num.get());
        entry.other.set(contract.sub.other.get());
        entry.word.set(contract.sub.word.get());
    }

    // test fixed arrays
    let mut arrays = contract.arrays;
    let mut slot = arrays.strings.setter(2).unwrap();
    slot.set_str("L2 is for you!");

    // test packed arrays
    for i in 0..5 {
        let mut slot = arrays.packed.get_mut(i).unwrap();
        slot.set(Uint::from(i));
    }

    // test arrays that don't fit into a single word
    for i in 0..2 {
        let mut slot = arrays.spill.get_mut(i).unwrap();
        slot.set(Address::with_last_byte(i as u8));
    }

    // test 2d arrays
    let mut matrix = arrays.matrix;
    for i in 0..4 {
        let mut inner = matrix.get_mut(i).unwrap();
        let mut slot = inner.get_mut(0).unwrap();
        slot.set(U8::from(i));

        let value = slot.get();
        let mut slot = inner.get_mut(1).unwrap();
        slot.set(value + U8::from(1));
    }

    // test vector of arrays
    for _ in 0..3 {
        let mut fixed = arrays.vector.grow();
        for i in 0..4 {
            let mut slot = fixed.get_mut(i).unwrap();
            slot.set(Signed::from_raw(Uint::from(i)));
        }
    }

    // test array of vectors
    for w in 0..4 {
        let mut vector = arrays.vectors.setter(w).unwrap();
        for i in 0..4 {
            vector.push(Signed::from_raw(Uint::from(i)));
        }
    }

    // test array of structs
    for i in 0..3 {
        let mut entry = arrays.structs.get_mut(i).unwrap();

        entry.num.set(contract.sub.num.get());
        entry.other.set(contract.sub.other.get());
        entry.word.set(contract.sub.word.get());
    }
}

fn remove(mut contract: Contract) {
    // pop all elements
    let mut bytes_full = contract.bytes_full;
    while let Some(value) = bytes_full.pop() {
        assert_eq!(value as usize, bytes_full.len());
    }
    assert!(bytes_full.is_empty());

    // pop until representation change
    let mut bytes_long = contract.bytes_long;
    while bytes_long.len() > 16 {
        assert!(bytes_long.pop().is_some());
    }

    // overwrite strings
    let mut chars = contract.chars;
    let spiders = r"/\oo/\ //\\(oo)//\\ /\oo/\";
    chars.set_str(spiders.repeat(6));
    chars.set_str("wasm is cute <3");

    // pop all elements
    let mut vector = contract.vector;
    while let Some(x) = vector.pop() {
        assert!(x == U64::from(vector.len()) || x == U64::from(77));
    }
    assert!(vector.is_empty() && vector.len() == 0);

    // erase inner vectors
    let mut nested = contract.nested;
    while nested.len() > 2 {
        nested.erase_last();
    }
    nested.shrink().map(|mut x| x.erase());

    // erase map elements
    let maps = contract.maps;
    let mut basic = maps.basic;
    for i in 0..7 {
        basic.delete(Uint::from(i));
    }
    let value = basic.take(Uint::from(7));
    assert_eq!(value, Address::with_last_byte(7));
    let value = basic.replace(Uint::from(8), Address::with_last_byte(32));
    assert_eq!(value, Address::with_last_byte(8));

    // erase vectors in map
    let mut vects = maps.vects;
    for a in 0..3 {
        let mut bools = vects.setter(Address::with_last_byte(a));
        bools.erase();
    }
    vects.delete(Address::with_last_byte(3));

    // erase a struct
    contract.structs.erase_last();

    // erase fixed arrays
    contract.arrays.matrix.erase();
    contract.arrays.vector.erase();
    contract.arrays.vectors.erase();
    contract.arrays.structs.erase();
}
