package main

import (
	"log"
	"strconv"
	"time"

	"github.com/StabbyCutyou/buffstreams"
	"github.com/StabbyCutyou/buffstreams/test/message"
	"github.com/golang/protobuf/proto"
)

// Test client to send a sample payload of data endlessly
// By default it points locally, but it can point to any network address
// TODO Make that externally configurable to make automating the test easier
func main() {
	cfg := &buffstreams.TCPConnConfig{
		MaxMessageSize: 2048,
		Address:        buffstreams.FormatAddress("127.0.0.1", strconv.Itoa(5031)),
	}
	ansBytes := make([]byte, 1024)
	name := "Stabby"
	date := time.Now().UnixNano()
	data := "This is an intenntionally long and rambling sentence to pad out the size of the message."
	msg := &message.Note{Name: &name, Date: &date, Comment: &data}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Print(err)
	}
	count := 0
	btw, err := buffstreams.DialTCP(cfg)
	if err != nil {
		log.Fatal(err)
	}
	currentTime := time.Now()
	lastTime := currentTime
	for {
		_, err := btw.Write(msgBytes)
		if err != nil {
			log.Print("There was an error")
			log.Print(err)
			break
		}
		count = count + 1
		if lastTime.Second() != currentTime.Second() {
			lastTime = currentTime
			log.Printf(", %d", count)
			count = 0
		}
		currentTime = time.Now()
		if n, err := btw.Read(ansBytes); err != nil {
			log.Println("Read error:", err)
			break
		} else {
			log.Println("From server:", string(ansBytes[0:n]))
		}
	}
}
