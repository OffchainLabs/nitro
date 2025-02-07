package arbutil

type FinalityData struct {
	FinalizedMsgCount    MessageIndex
	SafeMsgCount         MessageIndex
	ValidatedMsgCount    MessageIndex
	FinalitySupported bool
	BlockValidatorSet    bool
}
