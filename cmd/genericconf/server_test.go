package genericconf

import (
	"testing"
	"time"
)

func TestHTTPConfigDefault(t *testing.T) {
	if HTTPConfigDefault.Port != 8547 {
		t.Error("wrong port")
	}
}

func TestTimeoutConfig(t *testing.T) {
	config := HTTPServerTimeoutConfigDefault
	if config.ReadTimeout == 0 {
		t.Error("ReadTimeout should not be zero")
	}
}
