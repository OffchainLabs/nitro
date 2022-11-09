package util

var (
	_ = TimeReference(&realTimeReference{})
	_ = TimeReference(&ArtificialTimeReference{})
)
