package buffstreams

import "testing"

func TestNewBuffTCPListenerUsesDefaultMessageSize(t *testing.T) {
	cfg := BuffTCPListenerConfig{}
	buffM := NewBuffTCPListener(cfg)
	if buffM.maxMessageSize != DefaultMaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", DefaultMaxMessageSize, buffM.maxMessageSize)
	}
}

func TestNewBuffTCPListenerUsesSpecifiedMaxMessageSize(t *testing.T) {
	cfg := BuffTCPListenerConfig{
		MaxMessageSize: 8196,
	}
	buffM := NewBuffTCPListener(cfg)
	if buffM.maxMessageSize != cfg.MaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", cfg.MaxMessageSize, buffM.maxMessageSize)
	}
}
