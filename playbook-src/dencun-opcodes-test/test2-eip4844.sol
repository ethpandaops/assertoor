// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract EIP4844_Main {

    event Test1(uint indexed idx, bytes32 value);
    event Test2(int value);

    function test1() public {
        for(uint i = 0; i < 6; i++) {
            bytes32 bhash;
            assembly {
                bhash := blobhash(i)
            }
            emit Test1(i, bhash);
        }

        int blobfee;
        assembly {
            blobfee := blobbasefee()
        }
        emit Test2(blobfee);
    }
}
