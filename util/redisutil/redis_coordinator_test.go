// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package redisutil

import (
	"context"
	"sort"
	"testing"
	"time"
)

func TestRedisCoordinatorGetLiveliness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisUrl := CreateTestRedis(ctx, t)
	redisCoordinator, err := NewRedisCoordinator(redisUrl, 0)
	if err != nil {
		t.Fatalf("error creating redis coordinator: %s", err.Error())
	}
	wantLivelinessList := []string{"a", "b", "c", "d", "e", "f"}
	for _, url := range wantLivelinessList {
		if _, err = redisCoordinator.Client.Set(ctx, WantsLockoutKeyFor(url), WANTS_LOCKOUT_VAL, time.Minute).Result(); err != nil {
			t.Fatalf("error setting liveliness key for: %s err: %s", url, err.Error())
		}
	}
	haveLivelinessList, err := redisCoordinator.GetLiveliness(ctx)
	if err != nil {
		t.Fatalf("error getting liveliness list: %s", err.Error())
	}
	sort.Strings(haveLivelinessList)
	if len(wantLivelinessList) != len(haveLivelinessList) {
		t.Fatalf("liveliness list length mismatch")
	}
	for i, want := range wantLivelinessList {
		if haveLivelinessList[i] != want {
			t.Fatalf("liveliness list url mismatch. want: %s have: %s", want, haveLivelinessList[i])
		}
	}
}
