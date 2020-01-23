package main

import (
	"log"
	"fmt"
	"time"
	"net/http"

	"github.com/valyala/fasthttp"
)

type CloudEvent struct {
	Datacontenttype string `json:"datacontenttype"`
	Type	string `json:"type"`
	Time	time.Time `json:"time"`
	Id	string	`json:"id"`
	Source	string `json:"source"`
	Specversion	string `json:"specversion"`
	Data []byte `json:"data"`
}

func main(){
	listenAddr := "0.0.0.0:8080"
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		time, err := time.Parse(
        		time.RFC3339,
			string(ctx.Request.Header.Peek("ce-time")))
		if err != nil {
			ctx.Error("Invalid RFC3339 time", http.StatusBadRequest)
			return
		}
		event := CloudEvent{
			Datacontenttype: string(ctx.Request.Header.Peek("Content-Type")),
			Type: string(ctx.Request.Header.Peek("ce-type")),
			Time: time,
			Id: string(ctx.Request.Header.Peek("ce-id")),
			Source: string(ctx.Request.Header.Peek("ce-source")),
			Specversion: string(ctx.Request.Header.Peek("ce-specversion")),
			Data: ctx.PostBody(),
		}
		log.Println("%+v",event)
		fmt.Fprintf(ctx, "Hello, world! Requested path is %q", ctx.Path())
	}
	if err := fasthttp.ListenAndServe(listenAddr, requestHandler); err != nil {
		log.Fatalf("error in ListenAndServe: %s", err)
	}
}
