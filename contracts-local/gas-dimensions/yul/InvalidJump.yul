object "InvalidJump" {
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
            // bytes4(keccak("invalidJump()"))
            case 0xc59e9bfd {
                // Jump to position 1, which is never a JUMPDEST
                // Position 0 contains the PUSH instruction that pushes the function selector
                // Position 1 is in the middle of that instruction, making it an invalid jump destination
                let programCounter := 1
                verbatim_1i_0o(hex"600156", programCounter)
            }
            // Unknown selector (revert)
            default { revert(0, 0) }
        }
    }
}
