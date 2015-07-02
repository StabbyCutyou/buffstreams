package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/StabbyCutyou/buffstreams"
	"time"
)

func TestCallback(bts []byte) error {
	return nil
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	cfg := buffstreams.BuffManagerConfig{
		MaxMessageSize: 4096,
	}
	bm := buffstreams.New(cfg)
	bm.StartListening("5031", TestCallback)
	bm.DialOut("127.0.0.1", "5031")
	msg := []byte("HeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyH!")
	count := 0
	for {
		_, err := bm.WriteTo("127.0.0.1", "5031", msg, true)
		if err != nil {
			logrus.Error(err)
		}
		count = count + 1
		logrus.Print(count)
	}
	time.Sleep(time.Second * 10)
}
