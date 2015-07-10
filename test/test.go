package main

import (
	"github.com/StabbyCutyou/buffstreams"
	"github.com/StabbyCutyou/buffstreams/test/message"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"strconv"
	"time"
)

func TestCallback(bts []byte) error {
	return nil
}

// This is not a proper test, but it lets me benchmark how it
// performs on a raw level. Run this with the time command
func main() {
	cfg := buffstreams.BuffManagerConfig{
		MaxMessageSize: 256,
		EnableLogging:  true,
	}
	// 100 byte message
	name := "Stabby"
	date := time.Now().UnixNano()
	data := "This is an intenntionally long and rambling sentence to pad out the size of the message."
	msg := &message.Note{Name: &name, Date: &date, Comment: &data}
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Print(err)
	}
	startingPort := 5031
	bm := buffstreams.New(cfg)
	bm.StartListening(strconv.Itoa(startingPort), TestCallback)

	for i := 0; i < 10; i++ {
		go func(n int, port string, bm *buffstreams.BuffManager) {
			address := buffstreams.FormatAddress("127.0.0.1", port)
			count := 0
			for {
				_, err := bm.WriteTo(address, msgBytes, true)
				//log.Printf("Wrote %d Bytes of %d + 2, error was: %s", written, len(msgBytes), err)
				if err != nil {
					log.Printf("Error %s", err)
				}
				count = count + 1
				log.Printf("%d - %d", n, count)
			}
		}(i, strconv.Itoa(startingPort), bm)
		//startingPort = startingPort + 1
	}
	time.Sleep(time.Minute * 10)
}

func testTCPConns() {
	// Start Listening
	go func() {
		//readTicker := time.NewTicker(10 * time.Millisecond)
		tcpAddr, err := net.ResolveTCPAddr("tcp", ":5031")
		if err != nil {
			log.Print(err)
			return
		}
		receiveSocket, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Print(err)
			return
		}

		conn, err := receiveSocket.AcceptTCP()
		if err != nil {
			log.Printf("Error attempting to accept connection: %s", err)
		} else {
			buffer := make([]byte, 10)
			for {
				//select {
				// Check to see if we have a tick
				//case <-readTicker.C:
				log.Print("Trying to read")
				bytesRead, err := conn.Read(buffer)
				log.Printf("Bytes Read: %d out of %d", bytesRead, 10)
				if err != nil {
					log.Printf("Error: %s", err)
				}
				//}
			}
		}
	}()

	// Start writing
	go func() {
		writeTicker := time.NewTicker(10 * time.Second)
		tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:5031")
		if err != nil {
			log.Print(err)
			return
		}
		bytes := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		for {
			select {
			// Check to see if we have a tick
			case <-writeTicker.C:
				log.Printf("%s", time.Now())
				bytesWritten, err := conn.Write(bytes)
				log.Printf("Bytes Written: %d out of %d", bytesWritten, 10)
				if err != nil {
					log.Printf("Error: %s", err)
				}
			}
		}
	}()

	time.Sleep(time.Minute * 10)
}
