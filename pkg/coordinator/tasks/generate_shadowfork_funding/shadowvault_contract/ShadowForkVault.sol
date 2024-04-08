// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.0;

import "./BeaconChainProofs.sol";

contract ShadowForkVault {
    uint256 private originGenesisTime;
    bytes32[] private zeroHashes;
    mapping(address => uint256) private depositBalance;


    constructor(uint256 genesisTime) {
        originGenesisTime = genesisTime;

        zeroHashes = new bytes32[](33);
        for(uint256 i = 0; i < 32; i++) {
            bytes32 zeroHash = zeroHashes[i];
            zeroHashes[i+1] = sha256(abi.encodePacked(zeroHash, zeroHash));
        }
    }

    function getGenesisTime() public view returns (uint256) {
        return originGenesisTime;
    }

    function balanceof(address addr) public view returns (uint256) {
        return depositBalance[addr];
    }

    function getBeaconRootBySlot(uint256 slot) public view returns (bytes32) {
        uint256 slotTime = (slot * BeaconChainProofs.SECONDS_PER_SLOT) + originGenesisTime;
        return getBeaconRoot(slotTime);
    }

    function getBeaconRoot(uint256 time) public view returns (bytes32) {
        bytes32 result;
        (bool isSuccess, bytes memory response) = address(0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02).staticcall(abi.encodePacked(time + BeaconChainProofs.SECONDS_PER_SLOT));
        if(isSuccess) {
            assembly {
                result := mload(add(response, 32))
            }
        }
        return result;
    }

    // helper function to generate proofs
    // use fieldIndex = 0 for slot number proofs
    function generateHeaderProof(
        uint256 slotNumber,
        uint256 proposerIndex,
        bytes32 parentRoot,
        bytes32 stateRoot,
        bytes32 bodyRoot,
        uint256 fieldIndex
    ) public view returns (bytes memory) {
        (, bytes memory proof) = BeaconChainProofs.generateBlockRootProof(
            zeroHashes,
            slotNumber,
            proposerIndex,
            parentRoot,
            stateRoot,
            bodyRoot,
            fieldIndex
        );
        return proof;
    }

    receive() external payable {
        depositBalance[msg.sender] += msg.value;
    }

    function withdraw(uint256 amount) public {
        require(amount > 0, "amount must be greater than 0");
        require(depositBalance[msg.sender] >= amount, "amount exceeds balance");

        depositBalance[msg.sender] -= amount;

        (bool sent, ) = payable(msg.sender).call{value: amount}("");
        require(sent, "failed to send ether");
    }

    function shadowWithdraw(
        uint256 slotTime,
        uint256 slotNumber,
        bytes memory proof,
        address target,
        uint256 amount
    ) public {
        bytes32 blockRoot = getBeaconRoot(slotTime);
        require(blockRoot != bytes32(0), "no block root for slot time");

        bool proofValidity = BeaconChainProofs.verifySlotAgainstBlockRoot(blockRoot, slotNumber, proof);
        require(proofValidity, "block root verification failed");

        uint256 currentGenesisTime = slotTime - (slotNumber * BeaconChainProofs.SECONDS_PER_SLOT);
        require(currentGenesisTime > originGenesisTime, "not a shadow fork");

        require(address(this).balance >= amount, "amount exceeds balance");
        if (amount == 0) {
            amount = address(this).balance;
        }

        (bool sent, ) = payable(target).call{value: amount}("");
        require(sent, "failed to send ether");
    }
}