//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbstate

import "context"

type DataAvailabilityServiceReader interface {
	Retrieve(ctx context.Context, hash []byte) ([]byte, error)
}

const DASMessageHeaderFlag byte = 0x80

func IsDASMessageHeaderByte(header byte) bool {
	return (DASMessageHeaderFlag & header) > 0
}
