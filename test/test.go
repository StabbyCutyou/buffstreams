package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/StabbyCutyou/buffstreams"
	"time"
)

func TestCallback(bts []byte) error {
	logrus.Print("BYTES")
	logrus.Print(bts)
	logrus.Print(string(bts))
	return nil
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	bm := buffstreams.New()
	bm.StartListening("5031", TestCallback)
	bm.DialOut("127.0.0.1", "5031")
	msg := []byte("Hello!")
	for i := 0; i < 500; i++ {
		_, err := bm.WriteTo("127.0.0.1", "5031", msg, false)
		logrus.Info(err)
		time.Sleep(time.Millisecond * 10)
	}
	time.Sleep(time.Second * 10)
}
