// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IDepositContract {
    function deposit(
        bytes calldata pubkey,
        bytes calldata withdrawal_credentials,
        bytes calldata signature,
        bytes32 deposit_data_root
    ) external payable;
}

/// @notice Forwards a batch of deposits to the beacon-chain deposit contract in a single transaction.
/// @dev All deposits in a batch share the same withdrawal credentials and amount.
///      Each deposit must still carry its own pubkey, signature, and data root, so the
///      consensus layer is forced to verify every BLS signature individually.
contract BatchDeposit {
    IDepositContract public immutable depositContract;

    constructor(address _depositContract) {
        require(_depositContract != address(0), "BatchDeposit: zero deposit contract");
        depositContract = IDepositContract(_depositContract);
    }

    /// @param pubkeys              Concatenated BLS pubkeys, 48 bytes per deposit.
    /// @param signatures           Concatenated BLS signatures, 96 bytes per deposit.
    /// @param dataRoots            One deposit_data_root per deposit.
    /// @param withdrawalCredentials Shared 32-byte withdrawal credentials.
    /// @param amountWei            Amount (in wei) forwarded per deposit. msg.value must equal amountWei * dataRoots.length.
    function batchDeposit(
        bytes calldata pubkeys,
        bytes calldata signatures,
        bytes32[] calldata dataRoots,
        bytes calldata withdrawalCredentials,
        uint256 amountWei
    ) external payable {
        uint256 count = dataRoots.length;
        require(count > 0, "BatchDeposit: empty batch");
        require(pubkeys.length == count * 48, "BatchDeposit: bad pubkeys length");
        require(signatures.length == count * 96, "BatchDeposit: bad signatures length");
        require(withdrawalCredentials.length == 32, "BatchDeposit: bad creds length");
        require(msg.value == amountWei * count, "BatchDeposit: bad value");

        IDepositContract dc = depositContract;
        for (uint256 i = 0; i < count; ++i) {
            dc.deposit{value: amountWei}(
                pubkeys[i * 48:(i + 1) * 48],
                withdrawalCredentials,
                signatures[i * 96:(i + 1) * 96],
                dataRoots[i]
            );
        }
    }
}
