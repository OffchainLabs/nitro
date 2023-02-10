package solimpl

var (
	_ = selfInvalidator(&Assertion{})
	_ = selfInvalidator(&Challenge{})
	_ = selfInvalidator(&ChallengeVertex{})
)
