// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract Precompiles_Main {
    Precompiles_Child child;

    event Test1(uint8 indexed idx, bytes data);

    constructor() {
        child = new Precompiles_Child{salt: bytes32(uint256(1))}();
    }

    function test1() public {
        address res = ecrecover(
            0x345d9e6eb0778ac44a2803c061bf16a9cbd04495237b69fc85ad7ab2e256d9ee,
            0x000000000000000000000000000000000000000000000000000000000000001c,
            0x198177033ef6625421cd1b7ef6036264face53da5da4d7f2948aef3edf7e3f95,
            0x5c8fcf4db887386224512af70a8bc50d678069359c4d208a496d3a47339c7810
        );
        emit Test1(1, bytes.concat(bytes20(res)));
    }

    function test2() public {
        bytes32 res = sha256(abi.encodePacked(uint16(0x1337)));
        emit Test1(2, bytes.concat(res));
    }

    function test3() public {
        bytes32 res = ripemd160(abi.encodePacked(uint16(0x1337)));
        emit Test1(3, bytes.concat(res));
    }

    function test4() public {
        bytes32 data = bytes32(0x1337133713371337133713371337133713371337133713371337133713371337);
        bytes memory res = child.identity(bytes.concat(
            data
        ));
        emit Test1(4, res);
    }

    function test5() public {
        uint256 res = child.modExp(1337004242, 4242424242, 1337133713371337);
        emit Test1(5, bytes.concat(bytes32(res)));
    }

    function test9() public {
        uint32 rounds = 12;

        bytes32[2] memory h;
        h[0] = hex"48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5";
        h[1] = hex"d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b";

        bytes32[4] memory m;
        m[0] = hex"6162630000000000000000000000000000000000000000000000000000000000";
        m[1] = hex"0000000000000000000000000000000000000000000000000000000000000000";
        m[2] = hex"0000000000000000000000000000000000000000000000000000000000000000";
        m[3] = hex"0000000000000000000000000000000000000000000000000000000000000000";

        bytes8[2] memory t;
        t[0] = hex"03000000";
        t[1] = hex"00000000";

        bytes32[2] memory res = child.blake2F(rounds, h, m, t, true);
        emit Test1(9, bytes.concat(res[0], res[1]));
    }
}

contract Precompiles_Child {
    address public constant identityAddress =  0x0000000000000000000000000000000000000004;
    address public constant modExpAddress =    0x0000000000000000000000000000000000000005;
    address public constant blake2FAddress =   0x0000000000000000000000000000000000000009;

    function identity(bytes memory data) public returns (bytes memory) {
        bytes memory result = new bytes(data.length);
        assembly {
            let len := mload(data)
            if iszero(call(gas(), identityAddress, 0, add(data, 0x20), len, add(result,0x20), len)) {
                invalid()
            }
        }
        return result;
    }

    function modExp(uint256 _b, uint256 _e, uint256 _m) public returns (uint256 result) {
        assembly {
            // Free memory pointer
            let pointer := mload(0x40)
            // Define length of base, exponent and modulus. 0x20 == 32 bytes
            mstore(pointer, 0x20)
            mstore(add(pointer, 0x20), 0x20)
            mstore(add(pointer, 0x40), 0x20)
            // Define variables base, exponent and modulus
            mstore(add(pointer, 0x60), _b)
            mstore(add(pointer, 0x80), _e)
            mstore(add(pointer, 0xa0), _m)
            // Store the result
            let value := mload(0xc0)
            // Call the precompiled contract 0x05 = bigModExp
            if iszero(call(not(0), modExpAddress, 0, pointer, 0xc0, value, 0x20)) {
                revert(0, 0)
            }
            result := mload(value)
        }
    }

    function blake2F(uint32 rounds, bytes32[2] memory h, bytes32[4] memory m, bytes8[2] memory t, bool f) public view returns (bytes32[2] memory) {
        bytes32[2] memory output;
        bytes memory args = abi.encodePacked(rounds, h[0], h[1], m[0], m[1], m[2], m[3], t[0], t[1], f);
        assembly {
            if iszero(staticcall(not(0), blake2FAddress, add(args, 32), 0xd5, output, 0x40)) {
                revert(0, 0)
            }
        }
        return output;
    }

}
