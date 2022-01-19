
pragma solidity >=0.4.21 <0.9.0;

/**
* @title precompiled contract in every Arbitrum chain for retryable transaction related data retrieval and interactions. Exists at 0x000000000000000000000000000000000000006E
*/
interface ArbRetryableTx {

    /**
    * @notice Schedule an attempt to redeem a redeemable tx, donating all of the call's gas to the redeem.
    * Revert if ticketId does not exist.
    * @param ticketId unique identifier of retryable message: keccak256(keccak256(ArbchainId, inbox-sequence-number), uint(0) )
    * @return txId that the redeem attempt will have
     */
    function redeem(bytes32 ticketId) external returns(bytes32);

    /**
    * @notice Return the minimum lifetime of redeemable txn.
    * @return lifetime in seconds
    */
    function getLifetime() external view returns(uint);

    /**
    * @notice Return the timestamp when ticketId will age out, or zero if ticketId does not exist.
    * @param ticketId unique ticket identifier
    * @return timestamp for ticket's deadline
    */
    function getTimeout(bytes32 ticketId) external view returns(uint);

    /**
    * @notice Adds one lifetime period to the life of ticketId.
    * Donate gas to pay for the lifetime extension.
    * If successful, emits LifetimeExtended event.
    * Revert if ticketId does not exist, or if the timeout of ticketId is already at least one lifetime period in the future.
    * @param ticketId unique ticket identifier
    * @return new timeout of ticketId
    */
    function keepalive(bytes32 ticketId) external returns(uint);

    /**
    * @notice Return the beneficiary of ticketId.
    * Revert if ticketId doesn't exist.
    * @param ticketId unique ticket identifier
    * @return address of beneficiary for ticket
    */
    function getBeneficiary(bytes32 ticketId) external view returns (address);

    /**
    * @notice Cancel ticketId and refund its callvalue to its beneficiary.
    * Revert if ticketId doesn't exist, or if called by anyone other than ticketId's beneficiary.
    * @param ticketId unique ticket identifier
    */
    function cancel(bytes32 ticketId) external;

    event TicketCreated(bytes32 indexed ticketId);
    event LifetimeExtended(bytes32 indexed ticketId, uint newTimeout);
    event RedeemScheduled(bytes32 indexed ticketId, bytes32 indexed retryTxHash, uint64 sequenceNum, uint64 donatedGas, address gasDonor);
    event Redeemed(bytes32 indexed ticketId);
    event Canceled(bytes32 indexed ticketId);
}
