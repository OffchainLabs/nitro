package arbutil

type FinalityData struct {
	FinalizedMsgCount    MessageIndex
	SafeMsgCount         MessageIndex
	ValidatedMsgCount    MessageIndex
	FinalityNotSupported bool
	BlockValidatorSet    bool
}
