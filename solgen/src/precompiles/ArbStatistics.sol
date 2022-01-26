
pragma solidity >=0.4.21 <0.9.0;

interface ArbStatistics {
    // Get Arbitrum block number as well as the following statistics about the rollup right before the Nitro upgrade.
    //      Number of accounts
    //      Total storage allocated (includes storage that was later deallocated)
    //      Total ArbGas used
    //      Number of transaction receipt issued
    //      Number of contracts created
    function getStats() external view returns(uint, uint, uint, uint, uint, uint);
}
