package arbutil

type FinalityData struct {
	FinalizedMsgCount    MessageIndex
	SafeMsgCount         MessageIndex
	FinalityNotSupported bool
}
