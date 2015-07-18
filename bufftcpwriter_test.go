package buffstreams

import (
	"strconv"
	"testing"
	"time"

	"github.com/StabbyCutyou/buffstreams/test/message"
	"github.com/golang/protobuf/proto"
)

func exampleCallback(bts []byte) error {
	msg := &message.Note{}
	err := proto.Unmarshal(bts, msg)
	return err
}

var (
	writeConfig = BuffTCPWriterConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
		Address:        FormatAddress("127.0.0.1", strconv.Itoa(5031)),
	}

	listenConfig = BuffTCPListenerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
		Address:        FormatAddress("", strconv.Itoa(5031)),
		Callback:       exampleCallback,
	}

	btl = func() *BuffTCPListener {
		buffL := NewBuffTCPListener(listenConfig)
		buffL.StartListeningAsync()
		return buffL
	}()
	btw      = func() *BuffTCPWriter { buffT := NewBuffTCPWriter(writeConfig); buffT.Open(); return buffT }()
	name     = "Stabby"
	date     = time.Now().UnixNano()
	data     = "This is an intenntionally long and rambling sentence to pad out the size of the message."
	msg      = &message.Note{Name: &name, Date: &date, Comment: &data}
	msgBytes = func(*message.Note) []byte { b, _ := proto.Marshal(msg); return b }(msg)
)

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

func BenchmarkWrite(b *testing.B) {
	for n := 0; n < b.N; n++ {
		btw.Write(msgBytes)
	}
}
