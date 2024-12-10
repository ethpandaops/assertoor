// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract EIP7702_Main {

    function multiCall(address[] calldata targets, bytes[] calldata data) public  returns (bytes[] memory) {
        require(targets.length == data.length, "target length != data length");

        bytes[] memory results = new bytes[](data.length);

        for (uint256 i; i < targets.length; i++) {
            (bool success, bytes memory result) = targets[i].call(data[i]);
            require(success, "call failed");
            results[i] = result;
        }

        return results;
    }

    function test1(address wallet, address delegate) public payable {
        // test storage on wallet and delegate
        EIP7702_Delegate(wallet).testStorage(1, 1);
        EIP7702_Delegate(delegate).testStorage(2, 1);
        EIP7702_Delegate(wallet).testStorage(3, 2);
    }

    function test2(address wallet, address delegate) public payable {
        // test transient storage on wallet and delegate
        EIP7702_Delegate(wallet).testTransientStorage(1, 1);
        EIP7702_Delegate(delegate).testTransientStorage(2, 1);
        EIP7702_Delegate(wallet).testTransientStorage(3, 2);
    }

    function test3(address wallet) public payable {
        // test contract creation & destruction from wallet
        address child = EIP7702_Delegate(wallet).createChild{value: msg.value}();
        EIP7702_Delegate(wallet).destructChild(child);
        child = EIP7702_Delegate(wallet).create2Child{value: msg.value}(1);
        EIP7702_Delegate(wallet).destructChild(child);
    }

}

contract EIP7702_Delegate {
    uint storage1;

    function create(uint256 amount, bytes memory bytecode) private returns (address) {
        address addr;
        assembly {
            addr := create(amount, add(bytecode, 0x20), mload(bytecode))

            if iszero(extcodesize(addr)) {
                revert(0, 0)
            }
        }

        return addr;
    }

    function create2(uint256 amount, uint salt, bytes memory bytecode) private returns (address) {
        address addr;
        assembly {
            addr := create2(amount, add(bytecode, 0x20), mload(bytecode), salt)

            if iszero(extcodesize(addr)) {
                revert(0, 0)
            }
        }
        return addr;
    }

    function createChild() public payable returns (address) {
        bytes memory childCode = type(EIP7702_Child).creationCode;
        return create(msg.value, childCode);
    }

    function create2Child(uint salt) public payable returns (address) {
        bytes memory childCode = type(EIP7702_Child).creationCode;
        return create2(msg.value, salt, childCode);
    }

    function destructChild(address child) public {
        EIP7702_Child(child).destroy();
    }

    event Storage1Value(address indexed sender, address indexed origin, uint indexed index, uint value);
    event Storage2Value(address indexed sender, address indexed origin, uint indexed index, uint value);

    function testStorage(uint index, uint add) public payable {
        storage1 += add;
        emit Storage1Value(msg.sender, tx.origin, index, storage1);
    }

    function testTransientStorage(uint index, uint add) public payable {
        uint256 value;
        assembly {
            value := tload(0x00)
        }

        value += add;
        emit Storage2Value(msg.sender, tx.origin, index, value);

        assembly {
            tstore(0x00, value)
        }
    }
}

contract EIP7702_Child {
    event ChildCreated(address indexed sender, address indexed origin, address child);
    event ChildDestruct(address indexed sender, address indexed origin, address child);

    constructor() payable {
        emit ChildCreated(msg.sender, tx.origin, address(this));
    }

    function destroy() public {
        emit ChildDestruct(msg.sender, tx.origin, address(this));
        selfdestruct(payable(msg.sender));
    }

}
