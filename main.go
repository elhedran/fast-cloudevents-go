package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// CloudEvent represents a stored cloud event in memory.
type CloudEvent struct {
	Datacontenttype string            `json:"datacontenttype"`
	Type            string            `json:"type"`
	Time            time.Time         `json:"time"`
	Id              string            `json:"id"`
	Source          string            `json:"source"`
	Specversion     string            `json:"specversion"`
	Data            []byte            `json:"data"`
	Extensions      map[string]string `json:"extensions"`
}

// main shows an example implementation of each method with a fasthttp server.
func main() {
	listenAddr := "0.0.0.0:8080"
	requestHandler := func(ctx *fasthttp.RequestCtx) {
		event, err := FastHTTPToEventBinary(ctx)
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
		EventToFastHTTPBinary(event)(ctx) //fmt.Fprintf(ctx, "%q", ctx.Path())
	}
	if err := fasthttp.ListenAndServe(listenAddr, requestHandler); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

// FastHTTPToEventBinary returns an event from the given fasthttp request in binary mode.
func FastHTTPToEventBinary(ctx *fasthttp.RequestCtx) (ce CloudEvent, err error) {
	time, err := time.Parse(
		time.RFC3339,
		string(ctx.Request.Header.Peek("ce-time")))
	if err != nil {
		err = errors.New("Invalid RFC3339 time")
		return
	}
	ce = CloudEvent{
		Datacontenttype: string(ctx.Request.Header.Peek("Content-Type")),
		Type:            string(ctx.Request.Header.Peek("ce-type")),
		Time:            time,
		Id:              string(ctx.Request.Header.Peek("ce-id")),
		Source:          string(ctx.Request.Header.Peek("ce-source")),
		Specversion:     string(ctx.Request.Header.Peek("ce-specversion")),
		Data:            ctx.PostBody(),
		Extensions:      FastHTTPToExtensionsBinary(ctx),
	}
	return
}

// EventToFastHTTPBinary sets the given headers and body on the given fasthttp response in binary mode.
func EventToFastHTTPBinary(event CloudEvent) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Content-Type", event.Datacontenttype)
		ctx.Response.Header.Set("ce-type", event.Type)
		ctx.Response.Header.Set("ce-time", event.Time.Format(time.RFC3339))
		ctx.Response.Header.Set("ce-id", event.Id)
		ctx.Response.Header.Set("ce-source", event.Source)
		ctx.Response.Header.Set("ce-specversion", event.Specversion)
		ctx.Write(event.Data)
		ExtensionsToFastHTTPBinary(ctx, event.Extensions)
	}
}

// FastHTTPToExtensionsBinary returns a map of ce- prefixed extensions from the given fasthttp request in binary mode.
func FastHTTPToExtensionsBinary(ctx *fasthttp.RequestCtx) map[string]string {
	head := map[string]string{}
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		if knownHeader(string(key)) {
			return
		}
		if parts := strings.Split(strings.ToLower(string(key)), "ce-"); len(parts) >= 2 {
			head[strings.Join(parts[1:], "ce-")] = string(value)
		}
	})
	return head
}

// ExtensionsToFastHTTPBinary sets the given extension headers on the given fasthttp response in binary mode.
func ExtensionsToFastHTTPBinary(ctx *fasthttp.RequestCtx, extensions map[string]string) {
	for k, v := range extensions {
		ctx.Response.Header.Set(fmt.Sprintf("ce-%s", k), v)
	}
}

// knownHeader returns true when the given string is a known, ce- prefixed header key.
func knownHeader(h string) bool {
	known := []string{
		"ce-type",
		"ce-time",
		"ce-id",
		"ce-source",
		"ce-specversion",
	}
	for _, kh := range known {
		if kh == h {
			return true
		}
	}
	return false
}
