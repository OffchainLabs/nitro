package arbtest

import (
	"testing"
)

// #########################################################################################################
// #########################################################################################################
//	                                             CREATE & CREATE2
// #########################################################################################################
// #########################################################################################################
//
// CREATE and CREATE2 only have two permutations, whether or not you
// transfer value with the creation
// Paying vs NoTransfer (is ether value being sent with this call?)

func TestDimLogCreateNoTransfer(t *testing.T) { t.Fail() }

func TestDimLogCreatePaying(t *testing.T) { t.Fail() }

func TestDimLogCreate2NoTransfer(t *testing.T) { t.Fail() }

func TestDimLogCreate2Paying(t *testing.T) { t.Fail() }
