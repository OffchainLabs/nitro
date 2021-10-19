
pragma solidity >=0.4.21 <0.7.0;

interface ArbStatistics {
    // Get the following statistics for this chain:
    //      Number of Arbitrum blocks
    //      Number of accounts
    //      Total storage allocated (includes storage that was later deallocated)
    //      Total ArbGas used
    //      Number of transaction receipt issued
    //      Number of contracts created
    function getStats() external view returns(uint, uint, uint, uint, uint, uint);
}

