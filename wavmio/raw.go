package wavmio

func getLastBlockHash() [32]byte
func readInboxMessage([]byte) bool
func advanceInboxMessage()
func resolvePreImage(hash [32]byte, result []byte) bool
func setLastBlockHash([32]byte)
