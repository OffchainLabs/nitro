package assertionchain

var (
	_ = selfInvalidator(&Assertion{})
	_ = selfInvalidator(&Challenge{})
	_ = selfInvalidator(&ChallengeVertex{})
)
