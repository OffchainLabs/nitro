package genericconf

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
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

func TestReadHeaderTimeout(t *testing.T) {
	config := HTTPServerTimeoutConfigDefault

	// test ReadHeaderTimeout exists
	if config.ReadHeaderTimeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", config.ReadHeaderTimeout)
	}
}

func TestHTTPConfigApply(t *testing.T) {
	config := HTTPConfigDefault
	stackConf := &node.Config{}

	config.Apply(stackConf)

	if stackConf.HTTPPort != config.Port {
		t.Error("port not applied")
	}
	if stackConf.HTTPTimeouts.ReadTimeout != config.ServerTimeouts.ReadTimeout {
		t.Error("ReadTimeout not applied")
	}

}

func TestReadHeaderTimeoutApplied(t *testing.T) {
	config := HTTPConfigDefault
	stackConf := &node.Config{}

	config.Apply(stackConf)

	// ReadHeaderTimeout should now be applied
	if stackConf.HTTPTimeouts.ReadHeaderTimeout != config.ServerTimeouts.ReadHeaderTimeout {
		t.Error("ReadHeaderTimeout not applied correctly")
	}

}
