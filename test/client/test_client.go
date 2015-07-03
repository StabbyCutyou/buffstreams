package main

import (
	"github.com/StabbyCutyou/buffstreams"
	"github.com/StabbyCutyou/buffstreams/test/message"
	"github.com/golang/protobuf/proto"
	"log"
	"strconv"
	"time"
)

// Test client to send a sample payload of data endlessly
// By default it points locally, but it can point to any network address
// TODO Make that externally configurable to make automating the test easier
func main() {
	cfg := buffstreams.BuffManagerConfig{
		MaxMessageSize: 2048,
		EnableLogging:  true,
	}
	name := "Stabby"
	date := time.Now().UnixNano()
	data := "This is an intenntionally long and rambling sentence to pad out the size of the message."
	msg := &message.Note{Name: &name, Date: &date, Comment: &data}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Print(err)
	}
	count := 0
	bm := buffstreams.New(cfg)
	currentTime := time.Now()
	lastTime := currentTime
	for {
		_, err := bm.WriteTo("127.0.0.1", strconv.Itoa(5031), msgBytes, true)
		if err != nil {
			log.Print("EEEEEERRRRROOOOOOOORRRRRRRRRRR")
			log.Print(err)
		}
		count = count + 1
		if lastTime.Second() != currentTime.Second() {
			lastTime = currentTime
			log.Printf(", %d", count)
			count = 0
		}
		currentTime = time.Now()
	}
}
