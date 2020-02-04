package main

import (
	"log"
	"net/http"
	
	"github.com/valyala/fasthttp"

	"github.com/creativecactus/fast-cloudevents-go/fastce"
)

func main() {
	ExampleServer("0.0.0.0:8080")
}

/*
 ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗
 ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗
 ███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝
 ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗
 ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║
 ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝
*/

// ExampleServer shows an example implementation of each method with a fasthttp server.
func ExampleServer(listenAddr string) {
	router := func(ctx *fasthttp.RequestCtx) {
		switch p := string(ctx.Path()); p {
		case "/debug":
			ctx.Write([]byte("Hello World"))
			break
		default:
			ces, mode, err := fastce.GetEvents(ctx)
			if err != nil {
				log.Printf("ERR: %s", err.Error())
				ctx.Error(err.Error(), http.StatusBadRequest)
				return
			} else {
				log.Printf("OK : Received %d events in mode %d\n", len(ces), mode)
			}
			log.Printf("\tData: %#v\n", ces)
			fastce.PutEvents(ctx, ces, mode)
		}
	}
	log.Printf("Listening on %s", listenAddr)
	if err := fasthttp.ListenAndServe(listenAddr, router); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
