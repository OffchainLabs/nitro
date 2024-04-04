// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package jsonapi

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Uint64String is a uint64 that JSON marshals and unmarshals as string in decimal
type Uint64String uint64

func (u *Uint64String) UnmarshalJSON(b []byte) error {
	jsonString := string(b)
	if jsonString == "null" {
		return nil
	}

	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}

	*u = Uint64String(value)
	return nil
}

func (u Uint64String) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%d\"", uint64(u))), nil
}
