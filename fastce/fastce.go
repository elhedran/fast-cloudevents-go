package fastce

import (
	// "encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	j "github.com/creativecactus/fast-cloudevents-go/jsonce"

	"github.com/valyala/fasthttp"
)

// GetEvents determines the mode and content type of a request and gets any event(s) from it
func GetEvents(ctx *fasthttp.RequestCtx) (ces []j.CloudEvent, mode j.Mode, err error) {
	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	mode = GetMode(ctx)
	switch mode {
	case j.ModeBinary:
		ce, err := CtxBinaryToCE(ctx)
		if err !=nil {
			return ces, mode, fmt.Errorf("Could not get binary event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeStructure:
		// Determine the media type with which to parse the event
		// Or reject anything other than JSON
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
		ct := string(ctx.Request.Header.Peek("Content-Type"))
		if !strings.HasPrefix(ct, "application/cloudevents+json") {
			return ces, mode, fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ce, err := CtxStructureJSONToCE(ctx)
		if err !=nil {
			return ces, mode, fmt.Errorf("Could not get structure event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeBatch:
		err = fmt.Errorf("Unimplemented mode: %s", mode)
	default:
		err = fmt.Errorf("Unknown mode: %s", mode)
	}
	return
}
// CtxStructureJSONToCE converts a RequestCtx in Structured mode with JSON content type into a jsonce CloudEvent
func CtxStructureJSONToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	body := ctx.PostBody()
	ce = j.CloudEvent{}
	err = ce.UnmarshalJSON(body)
	if err != nil {
		return ce, fmt.Errorf("Could not unmarshal to event: %s", err.Error())
	}
	return ce, err
}

// CtxBinaryToCE converts a RequestCtx in Binary mode into a jsonce CloudEvent
func CtxBinaryToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	m := map[string]interface{}{}

	// Required
	m["id"] = string(ctx.Request.Header.Peek("ce-id"))
	m["source"] = string(ctx.Request.Header.Peek("ce-source"))
	m["specversion"] = string(ctx.Request.Header.Peek("ce-specversion"))
	m["type"] = string(ctx.Request.Header.Peek("ce-type"))

	// Optional
	m["datacontenttype"] = string(ctx.Request.Header.Peek("Content-Type"))
	m["dataschema"] = string(ctx.Request.Header.Peek("ce-dataschema"))
	m["subject"] = string(ctx.Request.Header.Peek("ce-subject"))
	m["time"] = string(ctx.Request.Header.Peek("ce-time"))

	// Additional
	j.SetData(m, ctx.PostBody())

	head := map[string]interface{}{}
	ctx.Request.Header.VisitAll(func(k, v []byte) {
		if !strings.HasPrefix(string(k),"ce-") {
			return
		}
		key := strings.TrimPrefix(string(k), "ce-")
		if j.InSlice(key, j.ContextProperties) {
			return
		}
		head[key] = v
	})
	m["extensions"] = head //FastHTTPToExtensionsBinary(ctx),

	ce = j.CloudEvent{}
	err = ce.FromMap(m)
	return ce, err
}

// GetMode uses the Content Type header to determine the content mode of the request
// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
func GetMode(ctx *fasthttp.RequestCtx) (mode j.Mode) {
	ct := string(ctx.Request.Header.Peek("Content-Type"))

	mode = j.ModeBinary
	if strings.HasPrefix(ct, "application/cloudevents-batch") {
		mode = j.ModeBatch
	} else if strings.HasPrefix(ct, "application/cloudevents") {
		mode = j.ModeStructure
	}
	return mode
}

// ExampleServer shows an example implementation of each method with a fasthttp server.
func ExampleServer(listenAddr string) {
	router := func(ctx *fasthttp.RequestCtx) {
		switch p := string(ctx.Path()); p {
			case "/debug":
				ctx.Write([]byte("Hello World"))
				break
			default:
				ces, mode, err := GetEvents(ctx)
				if err != nil {
					log.Printf("ERR: %s", err.Error())
					ctx.Error(err.Error(), http.StatusBadRequest)
					return
				} else {
					log.Printf("OK : Received %s events in mode %s", len(ces), mode)
				}
				fmt.Printf("\tData: %#v\n", ces)
				ctx.Write([]byte(fmt.Sprintf("STUB:NO REPLY: %#v", ces)))
		}
	}
	if err := fasthttp.ListenAndServe(listenAddr, router); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

// // EventToFastHTTPStructured sets the given headers and body on the given fasthttp response in structured mode.
// func EventToFastHTTPStructured(event j.CloudEvent) func(*fasthttp.RequestCtx)error {
// 	return func(ctx *fasthttp.RequestCtx) (err error) {
// 		ctx.Response.Header.Set("Content-Type", event.Datacontenttype)
// 	/*
// 		jsonEvent := map[string][]byte{}
// 		jsonEvent["type"] = []byte(event.Type)
// 		jsonEvent["time"] = []byte(event.Time.Format(time.RFC3339))
// 		jsonEvent["id"] = []byte(event.Id)
// 		jsonEvent["source"] = []byte(event.Source)
// 		jsonEvent["specversion"] = []byte(event.Specversion)
// 		for k, v := range event.Extensions {
// 			jsonEvent[k] = []byte(v)
// 		}
// 		jsonEvent["data"] = []byte(event.Data)
// 	*/	js, err := json.Marshal(event)
// 		if err != nil {
// 			err = fmt.Errorf("Could not marshal: %s", err.Error())
// 			return
// 		}
// 		ctx.Write(js)
// 		return
// 	}
// }

// EventToFastHTTPBinary sets the given headers and body on the given fasthttp response in binary mode.
func EventToFastHTTPBinary(event j.CloudEvent) func(*fasthttp.RequestCtx)error {
	return func(ctx *fasthttp.RequestCtx) (err error) {
		ctx.Response.Header.Set("Content-Type", event.Datacontenttype)
		ctx.Response.Header.Set("ce-type", event.Type)
		ctx.Response.Header.Set("ce-time", event.Time.Format(time.RFC3339Nano))
		ctx.Response.Header.Set("ce-id", event.Id)
		ctx.Response.Header.Set("ce-source", event.Source)
		ctx.Response.Header.Set("ce-specversion", event.Specversion)
		ctx.Write(event.Data)
		ExtensionsToFastHTTPBinary(ctx, event.Extensions)
		return
	}
}


// ExtensionsToFastHTTPBinary sets the given extension headers on the given fasthttp response in binary mode.
func ExtensionsToFastHTTPBinary(ctx *fasthttp.RequestCtx, extensions map[string]json.RawMessage) {
	for k, v := range extensions {
		ctx.Response.Header.Set(fmt.Sprintf("ce-%s", k), string(v))
	}
}
