package main

import (
	"fmt"
	"os"
	"testing"

	j "github.com/creativecactus/fast-cloudevents-go/jsonce"

	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	// Init
	listenAddr := "0.0.0.0:8080"
	server, errc := ExampleServer(listenAddr)

	// Run Tests
	result := m.Run()

	// Shutdown
	server.Shutdown()
	err := <-errc
	if err != nil {
		log.Fatalf("Server Error: %s", err)
	}
	os.Exit(result)
}
