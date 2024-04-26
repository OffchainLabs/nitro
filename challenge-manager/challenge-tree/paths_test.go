package challengetree

import (
	"context"
	"testing"
)

func TestIsConfirmableEssentialNode(t *testing.T) {
	p := &RoyalChallengeTree{}
	_, _, _ = p.IsConfirmableEssentialNode(context.Background(), isConfirmableArgs{})
}
