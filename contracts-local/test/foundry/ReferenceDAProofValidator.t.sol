// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.0;

import "forge-std/Test.sol";
import "../../src/osp/ReferenceDAProofValidator.sol";

contract ReferenceDAProofValidatorTest is Test {
    ReferenceDAProofValidator validator;
    uint256 constant PRIVATE_KEY =
        0x1234567890123456789012345678901234567890123456789012345678901234;
    address signer;

    function setUp() public {
        signer = vm.addr(PRIVATE_KEY);
        address[] memory trustedSigners = new address[](1);
        trustedSigners[0] = signer;
        validator = new ReferenceDAProofValidator(trustedSigners);
    }

    function buildValidProof(
        bytes memory preimage,
        uint256 offset
    ) internal view returns (bytes memory proof, bytes32 certHash) {
        bytes32 sha256Hash = sha256(abi.encodePacked(preimage));

        // Sign the hash
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, sha256Hash);

        // Create certificate: [header(1), sha256Hash(32), v(1), r(32), s(32)] = 98 bytes
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01; // header
        assembly {
            mstore(add(certificate, 33), sha256Hash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }
        certHash = keccak256(certificate);

        // Build proof with new format: [certSize(8), certificate(98), version(1), preimageSize(8), preimageData]
        uint256 proofLength = 8 + 98 + 1 + 8 + preimage.length;
        proof = new bytes(proofLength);

        // Set certificate size (8 bytes at position 0)
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        // Copy certificate (98 bytes starting at position 8)
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }

        // Set version (1 byte at position 106)
        proof[106] = bytes1(0x01);

        // Set preimage size (8 bytes at position 107)
        uint256 preimageLen = preimage.length;
        assembly {
            let preimageSize := shl(192, preimageLen)
            mstore(add(proof, 139), preimageSize)
        }

        // Copy preimage data (starting at position 115)
        for (uint256 i = 0; i < preimage.length; i++) {
            proof[115 + i] = preimage[i];
        }
    }

    function testValidateReadPreimage() public {
        // Test preimage data
        bytes memory preimage =
            "This is a test preimage that is longer than 32 bytes for testing chunk extraction";
        uint256 offset = 16; // Read from byte 16

        // Build valid proof
        (bytes memory proof, bytes32 certHash) = buildValidProof(preimage, offset);

        // Call validateReadPreimage
        bytes memory chunk = validator.validateReadPreimage(certHash, offset, proof);

        // Verify the chunk
        assertEq(chunk.length, 32, "Chunk should be 32 bytes");

        // Verify chunk contents match the expected slice of preimage
        for (uint256 i = 0; i < 32; i++) {
            if (offset + i < preimage.length) {
                assertEq(chunk[i], preimage[offset + i], "Chunk byte mismatch");
            } else {
                assertEq(chunk[i], 0, "Chunk padding should be zero");
            }
        }
    }

    function testValidateReadPreimageAtEnd() public {
        // Test reading at the end of preimage (less than 32 bytes available)
        bytes memory preimage = "Short preimage";
        uint256 offset = 8; // Only 6 bytes available from offset 8

        // Build valid proof
        (bytes memory proof, bytes32 certHash) = buildValidProof(preimage, offset);

        // Validate
        bytes memory chunk = validator.validateReadPreimage(certHash, offset, proof);

        // Should get "eimage" (6 bytes, no padding)
        assertEq(chunk.length, 6);
        assertEq(chunk[0], bytes1("e"));
        assertEq(chunk[1], bytes1("i"));
        assertEq(chunk[2], bytes1("m"));
        assertEq(chunk[3], bytes1("a"));
        assertEq(chunk[4], bytes1("g"));
        assertEq(chunk[5], bytes1("e"));
    }

    function testInvalidHash() public {
        bytes memory preimage = "Test preimage";
        bytes memory wrongPreimage = "Wrong preimage data";
        uint256 offset = 0;

        // Build a valid proof with the wrong preimage to get wrong hash in certificate
        (bytes memory proof, bytes32 certHash) = buildValidProof(wrongPreimage, offset);

        // Replace the preimage data with wrong preimage data
        // The preimage starts at offset 115 in the proof (after certSize(8) + certificate(98) + version(1) + preimageSize(8))
        for (uint256 i = 0; i < preimage.length; i++) {
            proof[115 + i] = preimage[i];
        }

        // Update preimage size to match the wrong preimage
        assembly {
            let preimageSize := shl(192, 13) // "Test preimage" is 13 bytes
            mstore(add(proof, 139), preimageSize)
        }

        // Should revert when preimage hash doesn't match
        vm.expectRevert("Invalid preimage hash");
        validator.validateReadPreimage(certHash, offset, proof);
    }

    function testInvalidVersion() public {
        bytes memory preimage = "Test";
        uint256 offset = 0;

        // Build a valid proof
        (bytes memory proof, bytes32 certHash) = buildValidProof(preimage, offset);

        // Set wrong version (version byte is at position 106)
        proof[106] = bytes1(0x02); // Wrong version

        vm.expectRevert("Unsupported proof version");
        validator.validateReadPreimage(certHash, offset, proof);
    }

    function testProofTooShort() public {
        // Create a proof that's too short to contain the certificate
        bytes memory proof = new bytes(40); // Has header but not enough for full certificate

        // Set certificate size to 98 at position 0
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        // Create a dummy certHash for the test
        bytes32 certHash = keccak256("test");

        vm.expectRevert("Proof too short for certificate");
        validator.validateReadPreimage(certHash, 0, proof);
    }

    function testCertificateHashMismatch() public {
        bytes memory preimage = "Test preimage";
        uint256 offset = 0;

        // Build a valid proof
        (bytes memory proof, bytes32 certHash) = buildValidProof(preimage, offset);

        // Use a different certHash than what's in the proof
        bytes32 wrongCertHash = keccak256("wrong certificate");

        vm.expectRevert("Certificate hash mismatch");
        validator.validateReadPreimage(wrongCertHash, offset, proof);
    }

    function testInvalidCertificateLength() public {
        // Build a proof with wrong certificate length
        bytes memory preimage = "Test";
        bytes32 sha256Hash = sha256(abi.encodePacked(preimage));

        // Create invalid certificate with wrong length (99 bytes instead of 98)
        bytes memory certificate = new bytes(99);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), sha256Hash)
        }
        // Add dummy signature bytes
        for (uint256 i = 33; i < 99; i++) {
            certificate[i] = bytes1(uint8(i));
        }

        bytes32 certHash = keccak256(certificate);

        // Build proof with invalid certificate
        uint256 proofLength = 8 + 99 + 1 + 8 + preimage.length;
        bytes memory proof = new bytes(proofLength);

        // Set certificate size
        assembly {
            let certSize := shl(192, 99) // Wrong size
            mstore(add(proof, 32), certSize)
        }

        // Copy certificate
        for (uint256 i = 0; i < 99; i++) {
            proof[8 + i] = certificate[i];
        }

        // Set version
        proof[107] = bytes1(0x01);

        // Set preimage size
        assembly {
            let preimageSize := shl(192, 4) // "Test" is 4 bytes
            mstore(add(proof, 140), preimageSize)
        }

        // Copy preimage
        for (uint256 i = 0; i < preimage.length; i++) {
            proof[116 + i] = preimage[i];
        }

        vm.expectRevert("Invalid certificate length");
        validator.validateReadPreimage(certHash, 0, proof);
    }

    function testInvalidCertificateHeader() public {
        // Build a proof with wrong certificate header
        bytes memory preimage = "Test";
        bytes32 sha256Hash = sha256(abi.encodePacked(preimage));

        // Sign the hash
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, sha256Hash);

        // Create certificate with wrong header
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x02; // Wrong header (should be 0x01)
        assembly {
            mstore(add(certificate, 33), sha256Hash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }

        bytes32 certHash = keccak256(certificate);

        // Build proof
        uint256 proofLength = 8 + 98 + 1 + 8 + preimage.length;
        bytes memory proof = new bytes(proofLength);

        // Set certificate size
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        // Copy certificate
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }

        // Set version
        proof[106] = bytes1(0x01);

        // Set preimage size
        assembly {
            let preimageSize := shl(192, 4)
            mstore(add(proof, 139), preimageSize)
        }

        // Copy preimage
        for (uint256 i = 0; i < preimage.length; i++) {
            proof[115 + i] = preimage[i];
        }

        vm.expectRevert("Invalid certificate header");
        validator.validateReadPreimage(certHash, 0, proof);
    }

    function testProofTooShortForCustomData() public {
        bytes memory preimage = "Test";
        bytes32 sha256Hash = sha256(abi.encodePacked(preimage));

        // Sign the hash
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, sha256Hash);

        // Create valid certificate
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), sha256Hash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }
        bytes32 certHash = keccak256(certificate);

        // Build proof that's too short for custom data (missing version and rest)
        bytes memory proof = new bytes(106); // Only has certSize + certificate

        // Set certificate size
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        // Copy certificate
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }

        vm.expectRevert("Proof too short for custom data");
        validator.validateReadPreimage(certHash, 0, proof);
    }

    function testInvalidProofLength() public {
        bytes memory preimage = "Test";
        (bytes memory proof, bytes32 certHash) = buildValidProof(preimage, 0);

        // Truncate the proof to make it invalid
        bytes memory truncatedProof = new bytes(proof.length - 2);
        for (uint256 i = 0; i < truncatedProof.length; i++) {
            truncatedProof[i] = proof[i];
        }

        vm.expectRevert("Invalid proof length");
        validator.validateReadPreimage(certHash, 0, truncatedProof);
    }

    function testValidateCertificate() public {
        // Build a valid certificate validity proof
        bytes32 dataHash = sha256("test data");
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, dataHash);

        // Create certificate
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), dataHash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }

        // Build proof: [certSize(8), certificate, claimedValid(1), version(1)]
        bytes memory proof = new bytes(8 + 98 + 1 + 1);

        // Set certificate size
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        // Copy certificate
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }

        // Set claimedValid (not checked here, used in validateAndCheckCertificate)
        proof[106] = bytes1(0x01);

        // Set version
        proof[107] = bytes1(0x01);

        // Should return true for valid certificate
        assertTrue(validator.validateCertificate(proof));
    }

    function testValidateCertificateInvalidSignature() public {
        // Build certificate with invalid signature
        bytes32 dataHash = sha256("test data");

        // Create certificate with invalid signature
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), dataHash)
        }
        // Invalid v, r, s values
        certificate[33] = 0x1b; // v

        // Build proof
        bytes memory proof = new bytes(8 + 98 + 1 + 1);
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }
        proof[106] = bytes1(0x01);
        proof[107] = bytes1(0x01);

        // Should return false for invalid signature
        assertFalse(validator.validateCertificate(proof));
    }

    function testValidateCertificateUntrustedSigner() public {
        // Build certificate signed by untrusted signer
        uint256 untrustedKey = 0x9999999999999999999999999999999999999999999999999999999999999999;
        bytes32 dataHash = sha256("test data");
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(untrustedKey, dataHash);

        // Create certificate
        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), dataHash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }

        // Build proof
        bytes memory proof = new bytes(8 + 98 + 1 + 1);
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }
        proof[106] = bytes1(0x01);
        proof[107] = bytes1(0x01);

        // Should return false for untrusted signer
        assertFalse(validator.validateCertificate(proof));
    }

    function testValidateCertificateWrongLength() public {
        // Build proof with wrong certificate length
        bytes memory proof = new bytes(8 + 50 + 1 + 1);

        // Set wrong certificate size
        assembly {
            let certSize := shl(192, 50)
            mstore(add(proof, 32), certSize)
        }

        // Should return false for wrong certificate length
        assertFalse(validator.validateCertificate(proof));
    }

    function testValidateCertificateWrongHeader() public {
        // Build certificate with wrong header
        bytes32 dataHash = sha256("test data");
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, dataHash);

        bytes memory certificate = new bytes(98);
        certificate[0] = 0x02; // Wrong header
        assembly {
            mstore(add(certificate, 33), dataHash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }

        // Build proof
        bytes memory proof = new bytes(8 + 98 + 1 + 1);
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }
        proof[106] = bytes1(0x01);
        proof[107] = bytes1(0x01);

        // Should return false for wrong header
        assertFalse(validator.validateCertificate(proof));
    }

    function testValidateCertificateProofTooShort() public {
        // Create a proof that's too short (less than 8 bytes)
        bytes memory proof = new bytes(7);

        vm.expectRevert("Proof too short");
        validator.validateCertificate(proof);
    }

    function testValidateCertificateProofTooShortForCertAndValidity() public {
        // Create a proof with cert size but not enough data for cert and validity proof
        bytes memory proof = new bytes(107); // 8 + 98 + 1, missing the version byte

        // Set certificate size to 98
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }

        vm.expectRevert("Proof too short for cert and validity");
        validator.validateCertificate(proof);
    }

    function testValidateCertificateInvalidProofVersion() public {
        // Build a valid certificate
        bytes32 dataHash = sha256("test data");
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(PRIVATE_KEY, dataHash);

        bytes memory certificate = new bytes(98);
        certificate[0] = 0x01;
        assembly {
            mstore(add(certificate, 33), dataHash)
        }
        certificate[33] = bytes1(v);
        assembly {
            mstore(add(certificate, 66), r)
            mstore(add(certificate, 98), s)
        }

        // Build proof with invalid version
        bytes memory proof = new bytes(8 + 98 + 1 + 1);
        assembly {
            let certSize := shl(192, 98)
            mstore(add(proof, 32), certSize)
        }
        for (uint256 i = 0; i < 98; i++) {
            proof[8 + i] = certificate[i];
        }
        proof[106] = bytes1(0x01); // claimedValid
        proof[107] = bytes1(0x02); // Wrong version (should be 0x01)

        vm.expectRevert("Invalid proof version");
        validator.validateCertificate(proof);
    }
}
