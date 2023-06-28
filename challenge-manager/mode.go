package challengemanager

type Mode uint8

const (
	// WatchTowerMode mode is the default mode for the challenge manager.
	// It will not trigger a challenge creation, but will agree if it agrees with assertions and log errors if it disagrees.
	WatchTowerMode Mode = iota
	// ResolveMode mode will not post assertion, but will confirm assertion, and this is useful to get the stake back.
	ResolveMode
	// DefensiveMode mode will not post assertion, but will post and open challenges if it disagrees with any assertions.
	DefensiveMode
	// MakeMode mode will perform everything, ranging from posting assertions to staking to challenging and confirming.
	MakeMode
)
