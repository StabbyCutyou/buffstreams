package main

import (
	"log"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hhy5861/buffstreams"
	"github.com/hhy5861/buffstreams/message"
)

func main() {
	cfg := &buffstreams.TCPConnConfig{
		MaxMessageSize: 2048,
		Address:        buffstreams.FormatAddress("127.0.0.1", strconv.Itoa(8090)),
	}

	name := "rate"
	date := time.Now().UnixNano()
	data := "{\"ip\": \"106.108.82.102\",\"uid\": \"4532\"}"
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
	for i := 0; i < 10; i++ {
		_, err := btw.Write(msgBytes)
		if err != nil {
			log.Print("There was an error")
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
