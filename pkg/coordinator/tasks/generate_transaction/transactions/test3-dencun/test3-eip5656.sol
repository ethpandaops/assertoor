// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract EIP5656_Main {

    function test1() public {
        assembly {
            // load contract code to memory
            codecopy(0, 0, codesize())
            mcopy(0, 32, 64)
            log1(0, 94, 0x01)
        }
    }
}
