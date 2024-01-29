// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract EIP4788_Main {
    EIP4788_Child child;

    event Test1(uint indexed idx, uint256 indexed time, bytes32 value);

    constructor() {
        child = new EIP4788_Child();
    }

    function getBeaconRoot(uint256 time) public view returns (bytes32) {
        return child.getBeaconRoot(time);
    }

    function test1() public {
        for(uint i = 1; i < 100; i++) {
            uint256 time = block.timestamp - (i * 12);
            bytes32 beaconRoot = getBeaconRoot(time);
            emit Test1(i, time, beaconRoot);
        }
    }
}

contract EIP4788_Child {
    function getBeaconRoot(uint256 time) public view returns (bytes32) {
        uint256 zero = 0;
        assembly {
			mstore(0, time)
			let ok := staticcall(gas(), 0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02, 0, 32, 0, 32)
			if iszero(ok) {
                mstore(0, zero)
			}
			return(0, 32)
		}
    }
}