// SPDX-License-Identifier: MIT
pragma solidity ^0.8.26;

contract Controller {
    address[] public registeredWallets;
    
    function registerWallet() external {
        registeredWallets.push(msg.sender);
    }
    
    function runTest() external {
        for(uint i = 0; i < registeredWallets.length; i++) {
            Delegate(registeredWallets[i]).runTest();
        }
    }
    
    function getRegisteredWallets() external view returns (address[] memory) {
        return registeredWallets;
    }
}

contract Delegate {
    function runTest() external {
        // Deploy and destroy dummy contract
        bytes memory code = type(Dummy).creationCode;
        address dummy;
        assembly {
            dummy := create(0, add(code, 0x20), mload(code))
            if iszero(extcodesize(dummy)) {
                revert(0, 0)
            }
        }
        
        Dummy(dummy).destroy();
    }
}

contract Dummy {
    event Created(address indexed creator);
    event Destroyed(address indexed destroyer);
    
    constructor() {
        emit Created(msg.sender);
    }
    
    function destroy() external {
        emit Destroyed(msg.sender);
        selfdestruct(payable(msg.sender));
    }
}