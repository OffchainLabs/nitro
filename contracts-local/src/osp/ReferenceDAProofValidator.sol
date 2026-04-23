// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro-contracts/blob/main/LICENSE.md
// SPDX-License-Identifier: BUSL-1.1

pragma solidity ^0.8.0;

import "@nitro-contracts/osp/ICustomDAProofValidator.sol";

/**
 * @title ReferenceDAProofValidator
 * @notice Reference implementation of a CustomDA proof validator
 */
contract ReferenceDAProofValidator is ICustomDAProofValidator {
    uint256 private constant CERT_SIZE_LEN = 8;
    uint256 private constant CLAIMED_VALID_LEN = 1;
    uint256 private constant VERSION_LEN = 1;
    uint256 private constant PREIMAGE_SIZE_LEN = 8;
    uint256 private constant CERT_HEADER = 0x01;
    uint256 private constant PROVIDER_TYPE = 0xFF;
    uint256 private constant CERT_TOTAL_LEN = 99;
    uint256 private constant PROOF_VERSION = 0x01;

    mapping(address => bool) public trustedSigners;

    constructor(
        address[] memory _trustedSigners
    ) {
        for (uint256 i = 0; i < _trustedSigners.length; i++) {
            trustedSigners[_trustedSigners[i]] = true;
        }
    }
    /**
     * @notice Validates a ReferenceDA proof and returns the preimage chunk
     * @param certHash The keccak256 hash of the certificate (from machine's proven state)
     * @param offset The offset into the preimage to read from (from machine's proven state)
     * @param proof The proof data: [certSize(8), certificate, version(1), preimageSize(8), preimageData]
     * @return preimageChunk The up to 32-byte chunk at the specified offset
     */

    function validateReadPreimage(
        bytes32 certHash,
        uint256 offset,
        bytes calldata proof
    ) external pure override returns (bytes memory preimageChunk) {
        // Extract certificate size from proof
        uint256 certSize = uint256(uint64(bytes8(proof[0:CERT_SIZE_LEN])));

        require(proof.length >= CERT_SIZE_LEN + certSize, "Proof too short for certificate");
        bytes calldata certificate = proof[CERT_SIZE_LEN:CERT_SIZE_LEN + certSize];

        // Verify certificate hash matches what OSP validated
        require(keccak256(certificate) == certHash, "Certificate hash mismatch");

        // Validate certificate format: [header(1), providerType(1), dataHash(32), v(1), r(32), s(32)] = 99 bytes
        // First byte must be 0x01 (CustomDA message header flag)
        // Second byte must be 0xFF (ReferenceDA provider type)
        require(certificate.length == CERT_TOTAL_LEN, "Invalid certificate length");
        require(certificate[0] == bytes1(uint8(CERT_HEADER)), "Invalid certificate header");
        require(certificate[1] == bytes1(uint8(PROVIDER_TYPE)), "Invalid provider type");

        // Custom data starts after certificate
        uint256 customDataStart = CERT_SIZE_LEN + certSize;
        require(
            proof.length >= customDataStart + VERSION_LEN + PREIMAGE_SIZE_LEN,
            "Proof too short for custom data"
        );

        // Verify version
        require(proof[customDataStart] == bytes1(uint8(PROOF_VERSION)), "Unsupported proof version");

        // Extract preimage size
        uint256 preimageSize = uint256(
            uint64(
                bytes8(
                    proof[
                        customDataStart + VERSION_LEN:
                            customDataStart + VERSION_LEN + PREIMAGE_SIZE_LEN
                    ]
                )
            )
        );

        require(
            proof.length >= customDataStart + VERSION_LEN + PREIMAGE_SIZE_LEN + preimageSize,
            "Invalid proof length"
        );

        // Extract and verify preimage against sha256sum in the certificate
        bytes calldata preimage = proof[
            customDataStart + VERSION_LEN + PREIMAGE_SIZE_LEN:
                customDataStart + VERSION_LEN + PREIMAGE_SIZE_LEN + preimageSize
        ];
        bytes32 dataHashFromCert = bytes32(certificate[2:34]);
        require(sha256(preimage) == dataHashFromCert, "Invalid preimage hash");

        // Extract chunk at offset, matching the behavior of other preimage types
        // Returns up to 32 bytes from the specified offset
        uint256 preimageEnd = offset + 32;
        if (preimageEnd > preimage.length) {
            preimageEnd = preimage.length;
        }

        if (offset >= preimage.length) {
            return new bytes(0);
        }

        return preimage[offset:preimageEnd];
    }

    /**
     * @notice Validates whether a certificate is well-formed and legitimate
     * @dev The proof format is: [certSize(8), certificate, claimedValid(1), validityProof...]
     *      For ReferenceDA, the validityProof is simply a version byte (0x01).
     *      Other DA providers can include more complex validity proofs after the claimedValid byte,
     *      such as cryptographic signatures, merkle proofs, or other verification data.
     *
     *      Return vs Revert behavior:
     *      - Reverts when:
     *        - Proof is malformed (checked in this function)
     *        - Provided cert matches proven hash in the instruction (checked in hostio)
     *        - Claimed valid but is invalid and vice versa (checked in hostio)
     *      - Returns false when:
     *        - Certificate is malformed, including wrong length
     *        - Signature is malformed
     *        - Signer is not a trustedSigner
     *      - Returns true when:
     *        - Signer is a trustedSigner and certificate is valid
     *
     * @param proof The proof data starting with [certSize(8), certificate, claimedValid(1), validityProof...]
     * @return isValid True if the certificate is valid, false otherwise
     */
    function validateCertificate(
        bytes calldata proof
    ) external view override returns (bool isValid) {
        // Extract certificate size
        require(proof.length >= CERT_SIZE_LEN, "Proof too short");

        uint256 certSize = uint256(uint64(bytes8(proof[0:CERT_SIZE_LEN])));

        // Check we have enough data for certificate and validity proof
        require(
            proof.length >= CERT_SIZE_LEN + certSize + CLAIMED_VALID_LEN + VERSION_LEN,
            "Proof too short for cert and validity"
        );

        bytes calldata certificate = proof[CERT_SIZE_LEN:CERT_SIZE_LEN + certSize];

        // Certificate format is: [header(1), providerType(1), dataHash(32), v(1), r(32), s(32)] = 99 bytes total
        // First byte must be 0x01 (CustomDA message header flag)
        // Second byte must be 0xFF (ReferenceDA provider type)
        // Note: We return false for invalid certificates instead of reverting
        // because the certificate is already onchain. An honest validator must be able
        // to win a challenge to prove that ValidatePreImage should return false
        // so that an invalid cert can be skipped.
        if (certificate.length != CERT_TOTAL_LEN) {
            return false; // Invalid certificate length
        }
        if (certificate[0] != bytes1(uint8(CERT_HEADER))) {
            return false; // Invalid certificate header
        }

        if (certificate[1] != bytes1(uint8(PROVIDER_TYPE))) {
            return false; // Invalid provider type
        }

        // Extract data hash and signature components
        bytes32 dataHash = bytes32(certificate[2:34]);
        uint8 v = uint8(certificate[34]);
        bytes32 r = bytes32(certificate[35:67]);
        bytes32 s = bytes32(certificate[67:99]);

        // Recover signer from signature
        address signer = ecrecover(dataHash, v, r, s);

        // Check if signature is valid (ecrecover returns 0 on invalid signature)
        if (signer == address(0)) {
            return false;
        }

        // Check if signer is trusted
        if (!trustedSigners[signer]) {
            return false;
        }

        // Check version byte at the end of the proof
        // Note: This is a deliberately simple example. A good rule of thumb is that
        // anything added to the proof beyond the isValid byte must not be able to cause both
        // true and false to be returned from this function, given the same certificate.
        uint8 version = uint8(proof[proof.length - VERSION_LEN]);
        require(version == PROOF_VERSION, "Invalid proof version");

        return true;
    }
}
