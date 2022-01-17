pragma solidity >=0.4.21 <0.9.0;

interface ArbosTest {
    // unproductively burns the amount of L2 ArbGas
    function burnArbGas(uint gasAmount) external pure;
}
