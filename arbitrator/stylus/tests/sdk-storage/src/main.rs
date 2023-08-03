// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::{
    alloy_primitives::{Address, Uint, B256, I32, U16, U256, U32, U64, U8},
    prelude::*,
    stylus_proc::sol_storage,
};

#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

stylus_sdk::entrypoint!(user_main);

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
    };

    pub struct Struct {
        uint16 num;
        int32 other;
        bytes32 word;
    };

    pub struct Maps {
        mapping(uint256 => address) basic;
        mapping(address => bool[]) vects;
        mapping(uint32 => address)[] array;
        mapping(bytes1 => mapping(bool => uint256)) nested;
        mapping(string => Struct) structs;
    };
}

fn user_main(_: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut contract = unsafe { Contract::new(U256::ZERO, 0) };

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

    // test bytes and strings (TODO: add compares and pops)
    let mut bytes_full = contract.bytes_full;
    let mut bytes_long = contract.bytes_long;
    let mut chars = contract.chars;
    for i in 0..31 {
        bytes_full.push(i);
    }
    for i in 0..34 {
        bytes_long.push(i);
    }
    for c in "arbitrum stylus".chars() {
        chars.push(c);
    }
    for i in 0..31 {
        assert_eq!(bytes_full.get(i), Some(i));
    }
    for i in 0..34 {
        let setter = bytes_long.get_mut(i).unwrap();
        assert_eq!(setter.get()[0], i);
    }
    assert_eq!(bytes_full.get(32), None);
    assert_eq!(bytes_long.get(34), None);

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
        map.insert(U32::from(i), Address::with_last_byte(i));
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
    
    Ok(vec![])
}
