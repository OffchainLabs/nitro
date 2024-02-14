package programs

import "testing"

func TestConstants(t *testing.T) {
	err := testConstants()
	if err != nil {
		t.Fatal(err)
	}
}
