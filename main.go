package main

import (
	"log"

	"github.com/creativecactus/fast-cloudevents-go/fastce"
)

func main() {
	// _ is server, which can be .Shutdown()
	_, errc, _, err := fastce.ExampleServer("0.0.0.0:8080", fastce.ExampleHandler)
	if err != nil {
		log.Fatalf("Server Init Error: %s", err)
	}
	err = <-errc
	if err != nil {
		log.Fatalf("Server Error: %s", err)
	}
}
