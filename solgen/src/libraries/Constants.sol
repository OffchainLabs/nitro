pragma solidity ^0.8.4;

// 90% of Geth's 128KB tx size limit, leaving ~13KB for proving
uint256 constant MAX_DATA_SIZE = 117964;

// seconds per block
uint256 constant BLOCK_TIME = 15;