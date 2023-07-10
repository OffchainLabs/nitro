// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#[link(wasm_import_module = "forward")]
extern "C" {
    /// Gets the ETH balance in wei of the account at the given address.
    /// The semantics are equivalent to that of the EVM’s [`BALANCE`] opcode.
    ///
    /// [`BALANCE`]: <https://www.evm.codes/#31>
    pub(crate) fn account_balance(address: *const u8, dest: *mut u8);

    /// Gets the code hash of the account at the given address. The semantics are equivalent
    /// to that of the EVM's [`EXT_CODEHASH`] opcode. Note that the code hash of an account without
    /// code will be the empty hash
    /// `keccak("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470`.
    ///
    /// [`EXT_CODEHASH`]: <https://www.evm.codes/#3F>
    pub(crate) fn account_codehash(address: *const u8, dest: *mut u8);

    /// Reads a 32-byte value from permanent storage. Stylus's storage format is identical to
    /// that of the EVM. This means that, under the hood, this hostio is accessing the 32-byte
    /// value stored in the EVM state trie at offset `key`, which will be `0` when not previously
    /// set. The semantics, then, are equivalent to that of the EVM's [`SLOAD`] opcode.
    ///
    /// [`SLOAD`]: <https://www.evm.codes/#54>
    pub(crate) fn account_load_bytes32(key: *const u8, dest: *mut u8);

    /// Stores a 32-byte value to permanent storage. Stylus's storage format is identical to that
    /// of the EVM. This means that, under the hood, this hostio is storing a 32-byte value into
    /// the EVM state trie at offset `key`. Furthermore, refunds are tabulated exactly as in the
    /// EVM. The semantics, then, are equivalent to that of the EVM's [`SSTORE`] opcode.
    ///
    /// [`SSTORE`]: <https://www.evm.codes/#55>
    pub(crate) fn account_store_bytes32(key: *const u8, value: *const u8);

    /// Gets the basefee of the current block. The semantics are equivalent to that of the EVM's
    /// [`BASEFEE`] opcode.
    ///
    /// [`BASEFEE`]: <https://www.evm.codes/#48>
    pub(crate) fn block_basefee(basefee: *mut u8);

    /// Gets the unique chain identifier of the Arbitrum chain. The semantics are equivalent to
    /// that of the EVM's [`CHAIN_ID`] opcode.
    ///
    /// [`CHAIN_ID`]: <https://www.evm.codes/#46>
    pub(crate) fn block_chainid(chainid: *mut u8);

    /// Gets the coinbase of the current block, which on Arbitrum chains is the L1 batch poster's
    /// address. This differs from Ethereum where the validator including the transaction
    /// determines the coinbase.
    pub(crate) fn block_coinbase(coinbase: *mut u8);

    /// Gets the gas limit of the current block. The semantics are equivalent to that of the EVM's
    /// [`GAS_LIMIT`] opcode. Note that as of the time of this writing, `evm.codes` incorrectly
    /// implies that the opcode returns the gas limit of the current transaction.  When in doubt,
    /// consult [`The Ethereum Yellow Paper`].
    ///
    /// [`GAS_LIMIT`]: <https://www.evm.codes/#45>
    /// [`The Ethereum Yellow Paper`]: <https://ethereum.github.io/yellowpaper/paper.pdf>
    pub(crate) fn block_gas_limit() -> u64;

    /// Gets a bounded estimate of the L1 block number at which the Sequencer sequenced the
    /// transaction. See [`Block Numbers and Time`] for more information on how this value is
    /// determined.
    ///
    /// [`Block Numbers and Time`]: <https://developer.arbitrum.io/time>
    pub(crate) fn block_number(number: *mut u8);

    /// Gets a bounded estimate of the Unix timestamp at which the Sequencer sequenced the
    /// transaction. See [`Block Numbers and Time`] for more information on how this value is
    /// determined.
    ///
    /// [`Block Numbers and Time`]: <https://developer.arbitrum.io/time>
    pub(crate) fn block_timestamp() -> u64;

    /// Calls the contract at the given address with options for passing value and to limit the
    /// amount of gas supplied. The return status indicates whether the call succeeded, and is
    /// nonzero on failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which can
    /// be read via the `read_return_data` hostio. The bytes are not returned directly so that the
    /// programmer can potentially save gas by choosing which subset of the return result they'd
    /// like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`CALL`] opcode, including callvalue
    /// stipends and the 63/64 gas rule. This means that supplying the `u64::MAX` gas can be used
    /// to send as much as possible.
    ///
    /// [`CALL`]: <https://www.evm.codes/#f1>
    pub(crate) fn call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        value: *const u8,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    /// Gets the address of the current program. The semantics are equivalent to that of the EVM's
    /// [`ADDRESS`] opcode.
    ///
    /// [`ADDRESS`]: <https://www.evm.codes/#30>
    pub(crate) fn contract_address(address: *mut u8);

    /// Deploys a new contract using the init code provided, which the EVM executes to construct
    /// the code of the newly deployed contract. The init code must be written in EVM bytecode, but
    /// the code it deploys can be that of a Stylus program. The code returned will be treated as
    /// WASM if it begins with the EOF-inspired header `0xEF000000`. Otherwise the code will be
    /// interpreted as that of a traditional EVM-style contract. See [`Deploying Stylus Programs`]
    /// for more information on writing init code.
    ///
    /// On success, this hostio returns the address of the newly created account whose address is
    /// a function of the sender and nonce. On failure the address will be `0`, `return_data_len`
    /// will store the length of the revert data, the bytes of which can be read via the
    /// `read_return_data` hostio. The semantics are equivalent to that of the EVM's [`CREATE`]
    /// opcode, which notably includes the exact address returned.
    ///
    /// [`Deploying Stylus Programs`]: <https://developer.arbitrum.io/TODO>
    /// [`CREATE`]: <https://www.evm.codes/#f0>
    pub(crate) fn create1(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );

    /// Deploys a new contract using the init code provided, which the EVM executes to construct
    /// the code of the newly deployed contract. The init code must be written in EVM bytecode, but
    /// the code it deploys can be that of a Stylus program. The code returned will be treated as
    /// WASM if it begins with the EOF-inspired header `0xEF000000`. Otherwise the code will be
    /// interpreted as that of a traditional EVM-style contract. See [`Deploying Stylus Porgrams`]
    /// for more information on writing init code.
    ///
    /// On success, this hostio returns the address of the newly created account whose address is a
    /// function of the sender, salt, and init code. On failure the address will be `0`,
    /// `return_data_len` will store the length of the revert data, the bytes of which can be read
    /// via the `read_return_data` hostio. The semantics are equivalent to that of the EVM's
    /// `[CREATE2`] opcode, which notably includes the exact address returned.
    ///
    /// [`Deploying Stylus Programs`]: <https://developer.arbitrum.io/TODO>
    /// [`CREATE2`]: <https://www.evm.codes/#f5>
    pub(crate) fn create2(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        salt: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );

    /// Delegate calls the contract at the given address, with the option to limit the amount of
    /// gas supplied. The return status indicates whether the call succeeded, and is nonzero on
    /// failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which
    /// can be read via the `read_return_data` hostio. The bytes are not returned directly so that
    /// the programmer can potentially save gas by choosing which subset of the return result
    /// they'd like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`DELEGATE_CALL`] opcode, including the
    /// 63/64 gas rule. This means that supplying `u64::MAX` gas can be used to send as much as
    /// possible.
    ///
    /// [`DELEGATE_CALL`]: <https://www.evm.codes/#F4>
    pub(crate) fn delegate_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    /// Emits an EVM log with the given number of topics and data, the first bytes of which should
    /// be the 32-byte-aligned topic data. The semantics are equivalent to that of the EVM's
    /// [`LOG0`], [`LOG1`], [`LOG2`], [`LOG3`], and [`LOG4`] opcodes based on the number of topics
    /// specified. Requesting more than `4` topics will induce a revert.
    ///
    /// [`LOG0`]: <https://www.evm.codes/#a0>
    /// [`LOG1`]: <https://www.evm.codes/#a1>
    /// [`LOG2`]: <https://www.evm.codes/#a2>
    /// [`LOG3`]: <https://www.evm.codes/#a3>
    /// [`LOG4`]: <https://www.evm.codes/#a4>
    pub(crate) fn emit_log(data: *const u8, len: usize, topics: usize);

    /// Gets the amount of gas left after paying for the cost of this hostio. The semantics are
    /// equivalent to that of the EVM's [`GAS`] opcode.
    ///
    /// [`GAS`]: <https://www.evm.codes/#5a>
    pub(crate) fn evm_gas_left() -> u64;

    /// Gets the amount of ink remaining after paying for the cost of this hostio. The semantics
    /// are equivalent to that of the EVM's [`GAS`] opcode, except the units are in ink. See
    /// [`Ink and Gas`] for more information on Stylus's compute pricing.
    ///
    /// [`GAS`]: <https://www.evm.codes/#5a>
    /// [`Ink and Gas`]: <https://developer.arbitrum.io/TODO>
    pub(crate) fn evm_ink_left() -> u64;

    /// The `arbitrum_main!` macro handles importing this hostio, which is required if the
    /// program's memory grows. Otherwise compilation through the `ArbWasm` precompile will revert.
    /// Internally the Stylus VM forces calls to this hostio whenever new WASM pages are allocated.
    /// Calls made voluntarily will unproductively consume gas.
    #[allow(dead_code)]
    pub(crate) fn memory_grow(pages: u16);

    /// Gets the address of the account that called the program. For normal L2-to-L2 transactions
    /// the semantics are equivalent to that of the EVM's [`CALLER`] opcode, including in cases
    /// arising from [`DELEGATE_CALL`].
    ///
    /// For L1-to-L2 retryable ticket transactions, the top-level sender's address will be aliased.
    /// See [`Retryable Ticket Address Aliasing`] for more information on how this works.
    ///
    /// [`CALLER`]: <https://www.evm.codes/#33>
    /// [`DELEGATE_CALL`]: <https://www.evm.codes/#f4>
    /// [`Retryable Ticket Address Aliasing`]: <https://developer.arbitrum.io/arbos/l1-to-l2-messaging#address-aliasing>
    pub(crate) fn msg_sender(sender: *mut u8);

    /// Get the ETH value in wei sent to the program. The semantics are equivalent to that of the
    /// EVM's [`CALLVALUE`] opcode.
    ///
    /// [`CALLVALUE`]: <https://www.evm.codes/#34>
    pub(crate) fn msg_value(value: *mut u8);

    /// Reads the program calldata. The semantics are equivalent to that of the EVM's
    /// [`CALLDATA_COPY`] opcode when requesting the entirety of the current call's calldata.
    ///
    /// [`CALLDATA_COPY`]: <https://www.evm.codes/#37>
    pub(crate) fn read_args(dest: *mut u8);

    /// Copies the bytes of the last EVM call or deployment return result. Reverts if out of
    /// bounds. The semantics are equivalent to that of the EVM's [`RETURN_DATA_COPY`] opcode.
    ///
    /// [`RETURN_DATA_COPY`]: <https://www.evm.codes/#3e>
    pub(crate) fn read_return_data(dest: *mut u8, offset: usize, size: usize) -> usize;

    /// Writes the final return data. If not called before the program exists, the return data will
    /// be 0 bytes long. Note that this hostio does not cause the program to exit, which happens
    /// naturally when the `arbitrum_main` entry-point returns.
    pub(crate) fn return_data(data: *const u8, len: usize);

    /// Returns the length of the last EVM call or deployment return result, or `0` if neither have
    /// happened during the program's execution. The semantics are equivalent to that of the EVM's
    /// [`RETURN_DATA_SIZE`] opcode.
    ///
    /// [`RETURN_DATA_SIZE`]: <https://www.evm.codes/#3d>
    pub(crate) fn return_data_size() -> usize;

    /// Static calls the contract at the given address, with the option to limit the amount of gas
    /// supplied. The return status indicates whether the call succeeded, and is nonzero on
    /// failure.
    ///
    /// In both cases `return_data_len` will store the length of the result, the bytes of which can
    /// be read via the `read_return_data` hostio. The bytes are not returned directly so that the
    /// programmer can potentially save gas by choosing which subset of the return result they'd
    /// like to copy.
    ///
    /// The semantics are equivalent to that of the EVM's [`STATIC_CALL`] opcode, including the
    /// 63/64 gas rule. This means that supplying `u64::MAX` gas can be used to send as much as
    /// possible.
    ///
    /// [`STATIC_CALL`]: <https://www.evm.codes/#FA>
    pub(crate) fn static_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    /// Gets the gas price in wei per gas, which on Arbitrum chains equals the basefee. The
    /// semantics are equivalent to that of the EVM's [`GAS_PRICE`] opcode.
    ///
    /// [`GAS_PRICE`]: <https://www.evm.codes/#3A>
    pub(crate) fn tx_gas_price(gas_price: *mut u8);

    /// Gets the price of ink in evm gas basis points. See [`Ink and Gas`] for more information on
    /// Stylus's compute-pricing model.
    ///
    /// [`Ink and Gas`]: <https://developer.arbitrum.io/TODO>
    pub(crate) fn tx_ink_price() -> u64;

    /// Gets the top-level sender of the transaction. The semantics are equivalent to that of the
    /// EVM's [`ORIGIN`] opcode.
    ///
    /// [`ORIGIN`]: <https://www.evm.codes/#32>
    pub(crate) fn tx_origin(origin: *mut u8);
}

#[allow(dead_code)]
#[link(wasm_import_module = "console")]
extern "C" {
    /// Prints a 32-bit floating point number to the console. Only available in debug mode with
    /// floating point enabled.
    pub(crate) fn log_f32(value: f32);

    /// Prints a 64-bit floating point number to the console. Only available in debug mode with
    /// floating point enabled.
    pub(crate) fn log_f64(value: f64);

    /// Prints a 32-bit integer to the console, which can be either signed or unsigned.
    /// Only available in debug mode.
    pub(crate) fn log_i32(value: i32);

    /// Prints a 64-bit integer to the console, which can be either signed or unsigned.
    /// Only available in debug mode.
    pub(crate) fn log_i64(value: i64);

    /// Prints a UTF-8 encoded string to the console. Only available in debug mode.
    pub(crate) fn log_txt(text: *const u8, len: usize);
}

/// Caches the length of the most recent EVM return data
pub(crate) static mut RETURN_DATA_SIZE: CachedOption<usize> = CachedOption::new(return_data_size);

/// Caches a value to avoid paying for hostio invocations.
pub(crate) struct CachedOption<T: Copy> {
    value: Option<T>,
    loader: unsafe extern "C" fn() -> T,
}

impl<T: Copy> CachedOption<T> {
    const fn new(loader: unsafe extern "C" fn() -> T) -> Self {
        let value = None;
        Self { value, loader }
    }

    pub(crate) fn set(&mut self, value: T) {
        self.value = Some(value);
    }

    pub(crate) fn get(&mut self) -> T {
        if let Some(value) = &self.value {
            return *value;
        }

        let value = unsafe { (self.loader)() };
        self.value = Some(value);
        value
    }
}
