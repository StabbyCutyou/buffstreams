package main

import (
	"github.com/StabbyCutyou/buffstreams"
	"log"
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
	}
	// 100 byte message
	msg := []byte("HeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyH!")
	startingPort := 5031
	for i := 0; i < 10; i++ {
		go func(n int, port string) {
			bm := buffstreams.New(cfg)
			bm.StartListening(strconv.Itoa(startingPort), TestCallback)
			count := 0
			for {
				_, err := bm.WriteTo("127.0.0.1", strconv.Itoa(startingPort), msg, true)
				if err != nil {
					log.Print("EEEEEERRRRROOOOOOOORRRRRRRRRRR")
					log.Print(err)
				}
				count = count + 1
				log.Printf("%d - %d", n, count)
			}
		}(i, strconv.Itoa(startingPort))
		startingPort = startingPort + 1
	}
	time.Sleep(time.Minute * 10)
}
