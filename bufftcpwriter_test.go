package buffstreams

import "testing"

func TestBuffTCPWriterUsesDefaultMessageSize(t *testing.T) {
	cfg := BuffTCPWriterConfig{}
	buffM := NewBuffTCPWriter(cfg)
	if buffM.maxMessageSize != DefaultMaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", DefaultMaxMessageSize, buffM.maxMessageSize)
	}
}

func TestBuffTCPWriterUsesSpecifiedMaxMessageSize(t *testing.T) {
	cfg := BuffTCPWriterConfig{
		MaxMessageSize: 8196,
	}
	buffM := NewBuffTCPWriter(cfg)
	if buffM.maxMessageSize != cfg.MaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", cfg.MaxMessageSize, buffM.maxMessageSize)
	}
}
