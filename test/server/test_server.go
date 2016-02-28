package main

import (
	"log"
	"strconv"
	"time"

	"github.com/StabbyCutyou/buffstreams"
	"github.com/StabbyCutyou/buffstreams/test/message"
	"github.com/golang/protobuf/proto"
)

// TestCallback is a simple server for test purposes. It has a single callback,
// which is to unmarshall some data and log it.
func (t *testController) TestCallback(bts []byte) error {
	msg := &message.Note{}
	err := proto.Unmarshal(bts, msg)
	if t.enableLogging {
		log.Print(msg.GetComment())
	}
	return err
}

type testController struct {
	enableLogging bool
}

func main() {
	tc := &testController{
		enableLogging: false,
	}
	cfg := buffstreams.TCPListenerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  tc.enableLogging,
		Address:        buffstreams.FormatAddress("", strconv.Itoa(5031)),
		Callback:       tc.TestCallback,
	}

	btl, err := buffstreams.ListenTCP(cfg)
	if err != nil {
		log.Print(err)
	}
	if err := btl.StartListening(); err != nil {
		log.Print(err)
	} else {
		// Need to block until ctrl+c, but having trouble getting signal trapping to work on OSX...
		time.Sleep(time.Minute * 10)
	}
}
