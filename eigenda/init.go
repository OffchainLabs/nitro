package eigenda

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var certDecodeABI abi.ABI

func init() {
	var err error
	certDecodeRawABI := `[
		{
			"type": "function",
			"name": "decodeCert",
			"inputs": [
				{
					"name": "cert",
					"type": "tuple",
					"internalType": "struct ISequencerInbox.DACert",
					"components": [
						{
							"name": "blobVerificationProof",
							"type": "tuple",
							"internalType": "struct EigenDARollupUtils.BlobVerificationProof",
							"components": [
								{
									"name": "batchId",
									"type": "uint32",
									"internalType": "uint32"
								},
								{
									"name": "blobIndex",
									"type": "uint32",
									"internalType": "uint32"
								},
								{
									"name": "batchMetadata",
									"type": "tuple",
									"internalType": "struct IEigenDAServiceManager.BatchMetadata",
									"components": [
										{
											"name": "batchHeader",
											"type": "tuple",
											"internalType": "struct IEigenDAServiceManager.BatchHeader",
											"components": [
												{
													"name": "blobHeadersRoot",
													"type": "bytes32",
													"internalType": "bytes32"
												},
												{
													"name": "quorumNumbers",
													"type": "bytes",
													"internalType": "bytes"
												},
												{
													"name": "signedStakeForQuorums",
													"type": "bytes",
													"internalType": "bytes"
												},
												{
													"name": "referenceBlockNumber",
													"type": "uint32",
													"internalType": "uint32"
												}
											]
										},
										{
											"name": "signatoryRecordHash",
											"type": "bytes32",
											"internalType": "bytes32"
										},
										{
											"name": "confirmationBlockNumber",
											"type": "uint32",
											"internalType": "uint32"
										}
									]
								},
								{
									"name": "inclusionProof",
									"type": "bytes",
									"internalType": "bytes"
								},
								{
									"name": "quorumIndices",
									"type": "bytes",
									"internalType": "bytes"
								}
							]
						},
						{
							"name": "blobHeader",
							"type": "tuple",
							"internalType": "struct IEigenDAServiceManager.BlobHeader",
							"components": [
								{
									"name": "commitment",
									"type": "tuple",
									"internalType": "struct BN254.G1Point",
									"components": [
										{
											"name": "X",
											"type": "uint256",
											"internalType": "uint256"
										},
										{
											"name": "Y",
											"type": "uint256",
											"internalType": "uint256"
										}
									]
								},
								{
									"name": "dataLength",
									"type": "uint32",
									"internalType": "uint32"
								},
								{
									"name": "quorumBlobParams",
									"type": "tuple[]",
									"internalType": "struct IEigenDAServiceManager.QuorumBlobParam[]",
									"components": [
										{
											"name": "quorumNumber",
											"type": "uint8",
											"internalType": "uint8"
										},
										{
											"name": "adversaryThresholdPercentage",
											"type": "uint8",
											"internalType": "uint8"
										},
										{
											"name": "confirmationThresholdPercentage",
											"type": "uint8",
											"internalType": "uint8"
										},
										{
											"name": "chunkLength",
											"type": "uint32",
											"internalType": "uint32"
										}
									]
								}
							]
						}
					]
				}
			],
			"outputs": [],
			"stateMutability": "nonpayable"
		}
	]
	`
	certDecodeABI, err = abi.JSON(bytes.NewReader([]byte(certDecodeRawABI)))
	if err != nil {
		panic(err)
	}
}
