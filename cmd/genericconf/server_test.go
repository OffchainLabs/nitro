package genericconf

import "testing"

func TestHTTPConfigDefault(t *testing.T) {
	if HTTPConfigDefault.Port != 8547 {
		t.Error("wrong port")
	}
}
