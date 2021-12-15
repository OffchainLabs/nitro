//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

abstract contract Cloneable {
	bool private isMasterCopy;

	constructor() {
		isMasterCopy = true;
	}

    function safeSelfDestruct(address payable dest) internal {
        require(!isMasterCopy, "NOT_CLONE");
        selfdestruct(dest);
    }
}
