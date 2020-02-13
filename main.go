package main

import (
	"log"
	"net/http"

	"github.com/valyala/fasthttp"

	"github.com/creativecactus/fast-cloudevents-go/fastce"
)

func main() {
	// _ is server, which can be .Shutdown()
	_, errc := ExampleServer("0.0.0.0:8080")
	err := <-errc
	if err != nil {
		log.Fatalf("Server Error: %s", err)
	}
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
func ExampleServer(listenAddr string) (*fasthttp.Server, <-chan error) {
	router := func(ctx *fasthttp.RequestCtx) {
		switch p := string(ctx.Path()); p {
		case "/info":
			ctx.Write([]byte("Example Server"))
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
			fastce.SetEvents(ctx, ces, mode)
		}
	}
	server := &fasthttp.Server{
		Handler: router,
	}
	shutdown := make(chan error)
	log.Printf("Listening on %s", listenAddr)
	go func() {
		err := server.ListenAndServe(listenAddr)
		shutdown <- err
	}()
	return server, shutdown
}
