object "Reader4844" {
    code {
        datacopy(0, dataoffset("runtime"), datasize("runtime"))
        return(0, datasize("runtime"))
    }
    object "runtime" {
        code {
            // This contract does not accept callvalue
            if callvalue() { revert(0, 0) }

            // Match against the keccak of the ABI function signature needed.
            switch shr(0xe0, calldataload(0))
            // bytes4(keccak("getDataHashes()"))
            case 0xe83a2d82 {
                let i := 0
                for { } true { }
                {
                    // DATAHASH opcode has hex value 0x49
                    let hash := verbatim_1i_1o(hex"49", i)
                    if iszero(hash) { break }
                    mstore(add(mul(i, 32), 64), hash)
                    i := add(i, 1)
                }
                mstore(0, 32)
                mstore(32, i)
                return(0, add(mul(i, 32), 64))
            }
            // bytes4(keccak("getBlobBaseFee()"))
            case 0x1f6d6ef7 {
                // BLOBBASEFEE opcode has hex value 0x4a
                let blobBasefee := verbatim_0i_1o(hex"4a")
                mstore(0, blobBasefee)
                return(0, 32)
            }
            // Unknown selector (revert)
            default { revert(0, 0) }
        }
    }
}
