package buffstreams

import (
	"log"
	"os"
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
	buffWriteConfig = TCPConnConfig{
		MaxMessageSize: 2048,
		Address:        FormatAddress("127.0.0.1", strconv.Itoa(5034)),
	}

	buffWriteConfig2 = TCPConnConfig{
		MaxMessageSize: 2048,
		Address:        FormatAddress("127.0.0.1", strconv.Itoa(5035)),
	}

	listenConfig = TCPListenerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
		Address:        FormatAddress("", strconv.Itoa(5033)),
		Callback:       exampleCallback,
	}

	listenConfig2 = TCPListenerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
		Address:        FormatAddress("", strconv.Itoa(5034)),
		Callback:       exampleCallback,
	}

	listenConfig3 = TCPListenerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
		Address:        FormatAddress("", strconv.Itoa(5035)),
		Callback:       exampleCallback,
	}

	btl      = &TCPListener{}
	btl2     = &TCPListener{}
	btl3     = &TCPListener{}
	btc      = &TCPConn{}
	btc2     = &TCPConn{}
	name     = "Stabby"
	date     = time.Now().UnixNano()
	data     = "This is an intenntionally long and rambling sentence to pad out the size of the message."
	msg      = &message.Note{Name: &name, Date: &date, Comment: &data}
	msgBytes = func(*message.Note) []byte { b, _ := proto.Marshal(msg); return b }(msg)
)

func TestMain(m *testing.M) {
	btl, err := ListenTCP(listenConfig)
	if err != nil {
		log.Fatal(err)
	}
	btl.StartListeningAsync()
	btl2, err = ListenTCP(listenConfig2)
	if err != nil {
		log.Fatal(err)
	}
	btl2.StartListeningAsync()
	btl3, err = ListenTCP(listenConfig3)
	if err != nil {
		log.Fatal(err)
	}
	btl3.StartListeningAsync()
	btc, err = DialTCP(&buffWriteConfig)
	if err != nil {
		log.Fatal(err)
	}
	btc2, err = DialTCP(&buffWriteConfig2)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestDialBuffTCPUsesDefaultMessageSize(t *testing.T) {
	cfg := TCPConnConfig{
		Address: buffWriteConfig.Address,
	}
	buffM, err := DialTCP(&cfg)
	if err != nil {
		t.Errorf("Failed to open connection to %d: %s", cfg.Address, err)
	}
	if buffM.maxMessageSize != DefaultMaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", DefaultMaxMessageSize, buffM.maxMessageSize)
	}
}

func TestDialBuffTCPUsesSpecifiedMaxMessageSize(t *testing.T) {
	cfg := TCPConnConfig{
		Address:        buffWriteConfig.Address,
		MaxMessageSize: 8196,
	}
	conn, err := DialTCP(&cfg)
	if err != nil {
		t.Errorf("Failed to open connection to %d: %s", cfg.Address, err)
	}
	if conn.maxMessageSize != cfg.MaxMessageSize {
		t.Errorf("Expected Max Message Size to be %d, actually got %d", cfg.MaxMessageSize, conn.maxMessageSize)
	}
}

func BenchmarkWrite(b *testing.B) {
	for n := 0; n < b.N; n++ {
		btc.Write(msgBytes)
	}
}

func BenchmarkWrite2(b *testing.B) {
	for n := 0; n < b.N; n++ {
		btc2.Write(msgBytes)
	}
}
