package containers

import (
	"testing"
)

func TestMap(t *testing.T) {
	var m Map[int, int]
	m.Store(13, 31)
	m.Store(11, 12)
	m.Store(11, 11)
	m.Store(42, 24)
	m.Delete(1) // Deleting non-existing key should be a no-op.
	m.Delete(42)

	for _, tc := range []struct {
		desc      string
		k, want   int
		wantFound bool
	}{
		{
			desc:      "key added once",
			k:         13,
			want:      31,
			wantFound: true,
		},
		{
			desc:      "key overwritten with different value",
			k:         11,
			want:      11,
			wantFound: true,
		},
		{
			desc: "key doesn't exist",
			k:    -1,
		},
		{
			desc: "key that was deleted",
			k:    42,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if got, found := m.Load(tc.k); found != tc.wantFound || got != tc.want {
				t.Errorf("Load(%v) = (%v %t) want (%v %t)", tc.k, got, found, tc.want, tc.wantFound)
			}
		})
	}

}
