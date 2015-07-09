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
		EnableLogging:  true,
	}
	// 100 byte message
	msg := []byte("HeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyHeyheyheyH!")
	startingPort := 5031
	bm := buffstreams.New(cfg)
	bm.StartListening(strconv.Itoa(startingPort), TestCallback)

	for i := 0; i < 10; i++ {
		go func(n int, port string, bm *buffstreams.BuffManager) {
			address := buffstreams.FormatAddress("127.0.0.1", port)
			count := 0
			for {
				_, err := bm.WriteTo(address, msg, true)
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
