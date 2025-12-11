// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//
// Based on https://github.com/andydotdev/omitlint

package a

import (
	"github.com/offchainlabs/nitro/linters/testdata/src/jsonneverempty/b"
)

type (
	aliasBasic           = int
	underlyingBasic      int
	aliasUnderlyingBasic = underlyingBasic

	aliasStruct           = struct{}
	structType            struct{}
	aliasUnderlyingStruct = structType

	aliasExternalStruct           = b.StructType
	underlyingExternalStruct      b.StructType
	aliasUnderlyingExternalStruct = underlyingExternalStruct
)

type Struct struct {
	// Basic types
	Bool    bool    `json:"bool,omitempty"`
	Int     int     `json:"int,omitempty"`
	Float32 float32 `json:"float32,omitempty"`
	String  string  `json:"string,omitempty"`

	// Other types that can be empty
	Slice      []string           `json:"slice,omitempty"`
	Pointer    *string            `json:"pointer,omitempty"`
	Map        map[any]structType `json:"map,omitempty"`
	Channel    chan structType    `json:"channel,omitempty"`
	Func       func()             `json:"func,omitempty"`
	Interface  interface{}        `json:"interface,omitempty"`
	EmptyArray [0]structType      `json:"empty-array,omitempty"`

	// Aliases of types that can be empty
	AliasBasic           aliasBasic           `json:"aliasbasic,omitempty"`
	UnderlyingBasic      underlyingBasic      `json:"underlyingbasic,omitempty"`
	AliasUnderlyingBasic aliasUnderlyingBasic `json:"aliasunderlyingbasic,omitempty"`

	// Types that can never be empty
	Array                         [2]bool                       `json:"array,omitempty"`                         // want `field 'Array' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	Struct                        structType                    `json:"struct,omitempty"`                        // want `field 'Struct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	AliasStruct                   aliasStruct                   `json:"aliasstruct,omitempty"`                   // want `field 'AliasStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	AliasUnderlyingStruct         aliasUnderlyingStruct         `json:"aliasunderlyingstruct,omitempty"`         // want `field 'AliasUnderlyingStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	ExternalStruct                b.StructType                  `json:"externalstruct,omitempty"`                // want `field 'ExternalStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	AliasExternalStruct           aliasExternalStruct           `json:"aliasexternalstruct,omitempty"`           // want `field 'AliasExternalStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	UnderlyingExternalStruct      underlyingExternalStruct      `json:"underlyingexternalstruct,omitempty"`      // want `field 'UnderlyingExternalStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`
	AliasUnderlyingExternalStruct aliasUnderlyingExternalStruct `json:"aliasunderlyingexternalstruct,omitempty"` // want `field 'AliasUnderlyingExternalStruct' is marked 'omitempty', but it can never be empty; consider making it a pointer`

	// We ignore unexported fields, even if they have the incorrect `omitempty` tag
	unexported structType `json:"unexported,omitempty"`
}
