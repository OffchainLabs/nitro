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

func TestReadHeaderTimeoutNotApplied(t *testing.T) {
	config := HTTPConfigDefault
	stackConf := &node.Config{}
	
	config.Apply(stackConf)
	
	// ReadHeaderTimeout is not being applied due to TODO
	if stackConf.HTTPTimeouts.ReadHeaderTimeout == config.ServerTimeouts.ReadHeaderTimeout {
		t.Error("ReadHeaderTimeout should not be applied yet")
	}
}
