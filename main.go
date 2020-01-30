package main

import (
	"github.com/creativecactus/fast-cloudevents-go/fastce"

	"encoding/json"
	"log"
	"net/http"

	"github.com/valyala/fasthttp"
)

func main() {
	ExampleServer()
}

// ExampleServer shows an example implementation of each method with a fasthttp server.
func ExampleServer() {
	listenAddr := "0.0.0.0:8080"
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		event, err := fastce.FastHTTPToEventBinary(ctx)
		if err != nil {
			ctx.Error(err.Error(), http.StatusBadRequest)
			return
		}
		js, err := json.Marshal(event)
		if err != nil {
			ctx.Error("Could not marshal", http.StatusInternalServerError)
			return
		}
		log.Printf("%s\n", string(js))
		if err = json.Unmarshal(js, &event); err != nil {
			ctx.Error("Could not unmarshal", http.StatusInternalServerError)
			return
		}
		fastce.EventToFastHTTPBinary(event)(ctx) //fmt.Fprintf(ctx, "%q", ctx.Path())
	}
	if err := fasthttp.ListenAndServe(listenAddr, requestHandler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
