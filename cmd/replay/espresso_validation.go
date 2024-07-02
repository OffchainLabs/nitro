package main

import (
	"fmt"

	"github.com/offchainlabs/nitro/arbos"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/wavmio"
)

// / Handle Espresso Pre Conditions
// /
// / Description: This function takes in the most recently recieved message (of type arbostypes.MessageWithMetadata),
// /	 			  and a boolean from the ChainConfig
// / 			 and uses the parameters to perform all checks gating the modified STF logic in the arbitrum validator.
// /
// / Return: A boolean representing the result of a logical and of all checks (that don't result in panics) gating espressos STF logic.
// /
// / Panics: This function will panic if the message type is not an espresso message, but the HotShot height is non zero
// / 		as this is an invalid state for the STF to reside in.
// /
func handleEspressoPreConditions(message *arbostypes.MessageWithMetadata, isEnabled bool) bool {
	//calculate and cache all values needed to determine if the preconditions are met to enter the Espresso STF logic
	isNonEspressoMessage := arbos.IsL2NonEspressoMsg(message.Message)
	isEspressoMessage := !isNonEspressoMessage
	hotShotHeight := wavmio.GetEspressoHeight()
	validatingEspressoLivenessFailure := isNonEspressoMessage && (isEnabled || hotShotHeight != 0)
	validatingAgainstEspresso := isEspressoMessage && isEnabled

	// If conditions are such that we have been working in espresso mode, but we are suddenly receiving non espresso messages, something wrong
	// something incorrect has occured and we must panic
	if validatingEspressoLivenessFailure {
		l1Block := message.Message.Header.BlockNumber
		if wavmio.IsHotShotLive(l1Block) {
			panic(fmt.Sprintf("getting the centralized message while hotshot is good, l1Height: %v", l1Block))
		}
		panic("The messaged recieved by the STF is not an Espresso message, but the validator is running in Espresso mode")
	}
	return validatingAgainstEspresso
}
