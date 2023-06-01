package time

var (
	_ = Reference(&realTimeReference{})
	_ = Reference(&ArtificialTimeReference{})
)
