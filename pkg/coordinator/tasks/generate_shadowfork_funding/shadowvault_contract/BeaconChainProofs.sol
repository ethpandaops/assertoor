// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "./Merkle.sol";

//Utility library for parsing and PHASE0 beacon chain block headers
//SSZ Spec: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#merkleization
//BeaconBlockHeader Spec: https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
library BeaconChainProofs {
    // constants are the number of fields and the heights of the different merkle trees used in merkleizing beacon chain containers
    uint256 internal constant BEACON_BLOCK_HEADER_FIELD_TREE_HEIGHT = 3;

    uint256 internal constant BEACON_BLOCK_HEADER_FIELD_COUNT = 5;

    // in beacon block header https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
    uint256 internal constant SLOT_INDEX = 0;

    /// @notice The number of seconds in a slot in the beacon chain
    uint64 internal constant SECONDS_PER_SLOT = 12;

    /**
     * @notice This function verifies the slot number against the block root.
     * @param slotNumber is the beacon chain slot number to be proven against.
     * @param proof is the provided merkle proof
     * @param blockRoot is hashtree root of the latest block header in the beacon state
     */
    function verifySlotAgainstBlockRoot(
        bytes32 blockRoot,
        uint256 slotNumber,
        bytes memory proof
    ) internal view returns (bool) {
        if (proof.length != 32 * (BEACON_BLOCK_HEADER_FIELD_TREE_HEIGHT)) {
            return false;
        }

        return Merkle.verifyInclusionSha256({
            proof: proof,
            root: blockRoot,
            leaf: bytes32(reverse(slotNumber)),
            index: SLOT_INDEX
        });
    }

    function generateBlockRootProof(
        bytes32[] storage zeroHashes,
        uint256 slotNumber,
        uint256 proposerIndex,
        bytes32 parentRoot,
        bytes32 stateRoot,
        bytes32 bodyRoot,
        uint256 fieldIdx
    ) internal view returns (bytes32, bytes memory) {
        bytes32[] memory headerFieldRoots = new bytes32[](BEACON_BLOCK_HEADER_FIELD_COUNT);

        headerFieldRoots[0] = bytes32(reverse(slotNumber));
        headerFieldRoots[1] = bytes32(reverse(proposerIndex));
        headerFieldRoots[2] = parentRoot;
        headerFieldRoots[3] = stateRoot;
        headerFieldRoots[4] = bodyRoot;

        bytes32[][] memory tree = buildHashTree(zeroHashes, headerFieldRoots, BEACON_BLOCK_HEADER_FIELD_TREE_HEIGHT);

        bytes memory proof = buildProofFromTree(zeroHashes, tree, BEACON_BLOCK_HEADER_FIELD_TREE_HEIGHT, fieldIdx);
        bytes32 root = buildRootFromTree(tree);

        return (root, proof);
    }

    function buildHashTree(
        bytes32[] storage zeroHashes,
        bytes32[] memory values,
        uint256 layers
    ) internal view returns (bytes32[][] memory) {
        bytes32[][] memory tree = new bytes32[][](layers + 1);
        tree[0] = values;

        for(uint256 l = 0; l < layers; l++) {
            uint256 layerSize = tree[l].length;
            uint256 paddedLayerSize;
            if (layerSize % 2 == 1) {
                paddedLayerSize = layerSize + 1;
            } else {
                paddedLayerSize = layerSize;
            }

            uint256 nextLevelSize = paddedLayerSize / 2;
            bytes32[] memory nextValues = new bytes32[](nextLevelSize);

            for (uint256 i = 0; i < paddedLayerSize; i += 2) {
                bytes32 leftHash = tree[l][i];
                bytes32 rightHash;
                
                if(i+1 >= layerSize) {
                    rightHash = zeroHashes[l];
                } else {
                    rightHash = tree[l][i+1];
                }

                nextValues[i/2] = sha256(abi.encodePacked(leftHash, rightHash));
            }

            tree[l+1] = nextValues;
        }

        return tree;
    }

    function buildProofFromTree(
        bytes32[] memory zeroHashes,
        bytes32[][] memory tree,
        uint256 layers,
        uint256 index
    ) internal pure returns (bytes memory) {
        bytes32[] memory proof = new bytes32[](layers);
        for(uint256 l = 0; l < layers; l++) {

            uint256 layerIndex = (index / (2**l))^1;
            if(layerIndex < tree[l].length) {
                proof[l] = tree[l][layerIndex];
            } else {
                proof[l] = zeroHashes[l];
            }
        }

        bytes memory result = abi.encodePacked(proof[0]);
        for(uint256 l = 1; l < layers; l++) {
            result = bytes.concat(result, abi.encodePacked(proof[l]));
        }
        
        return result;
    }

    function buildRootFromTree(
        bytes32[][] memory tree
    ) internal pure returns (bytes32) {
        uint256 treeSize = tree.length;
        return tree[treeSize-1][0];
    }

    function reverse(uint256 input) internal pure returns (uint256 v) {
        v = input;

        // swap bytes
        v = ((v & 0xFF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00) >> 8) |
            ((v & 0x00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF00FF) << 8);

        // swap 2-byte long pairs
        v = ((v & 0xFFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000) >> 16) |
            ((v & 0x0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF0000FFFF) << 16);

        // swap 4-byte long pairs
        v = ((v & 0xFFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000) >> 32) |
            ((v & 0x00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF00000000FFFFFFFF) << 32);

        // swap 8-byte long pairs
        v = ((v & 0xFFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF0000000000000000) >> 64) |
            ((v & 0x0000000000000000FFFFFFFFFFFFFFFF0000000000000000FFFFFFFFFFFFFFFF) << 64);

        // swap 16-byte long pairs
        v = (v >> 128) | (v << 128);
    }

}