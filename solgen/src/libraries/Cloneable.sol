//SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

abstract contract Cloneable {
	bool internal isMasterCopy;

	constructor() {
		isMasterCopy = true;
	}
}
