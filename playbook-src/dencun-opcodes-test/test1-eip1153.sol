// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract EIP1153_Main {
    EIP1153_Child1 child1;

    event Test2(uint8 indexed idx, uint256 value);

    constructor() {
        child1 = new EIP1153_Child1{salt: bytes32(uint256(1))}();

        // check transient storage across 2 calls
        child1.inc(1);
        child1.inc(2);

        // shouldn't affect transient storage of this contract context
        uint256 value;
        assembly {
            value := tload(0x00)
        }
        require(value == 0, "transient storage pollution");
    }

    function test1() public {
        child1.inc(3);
        child1.inc(4);

        // shouldn't affect transient storage of this contract context
        uint256 value;
        assembly {
            value := tload(0x00)
        }
        require(value == 0, "transient storage pollution");
    }

    function test2() public {
        child1.inc(5);
    }
}

contract EIP1153_Child1 {
    event Test1(uint8 indexed idx, uint256 value);

    function inc(uint8 idx) public {
        uint256 value;
        assembly {
            value := tload(0x00)
        }

        value++;

        emit Test1(idx, value);

        assembly {
            tstore(0x00, value)
        }
    }

}
