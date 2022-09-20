package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	nUri := os.Getenv("NATS_URI")
	if strings.EqualFold(nUri, "") {
		nUri = "nats://localhost:4222"
	}

	var err error

	nc, err := nats.Connect(nUri)

	if err != nil {
		log.Fatal("Error establishing connection to NATS:", err)
	}

	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("cmd/producer/model%d.json", i)
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("Error getting file data:", err)
		}

		err = nc.Publish("orders", data)
		if err != nil {
			log.Fatal("Failed to send data to chanel:", err)
		}
		err = nc.Flush()
		if err != nil {
			log.Fatal("Failed to flush data to chanel:", err)
		}

		fmt.Printf("send model %d \n", i)
		time.Sleep(time.Second * 5)
	}

}
