package fastce

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	j "github.com/creativecactus/fast-cloudevents-go/jsonce"

	"github.com/valyala/fasthttp"
)

/*
  ██████╗ ██████╗ ███╗   ██╗███████╗██╗   ██╗███╗   ███╗███████╗
 ██╔════╝██╔═══██╗████╗  ██║██╔════╝██║   ██║████╗ ████║██╔════╝
 ██║     ██║   ██║██╔██╗ ██║███████╗██║   ██║██╔████╔██║█████╗
 ██║     ██║   ██║██║╚██╗██║╚════██║██║   ██║██║╚██╔╝██║██╔══╝
 ╚██████╗╚██████╔╝██║ ╚████║███████║╚██████╔╝██║ ╚═╝ ██║███████╗
  ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚═╝     ╚═╝╚══════╝
*/

// GetEvents determines the mode and content type of a request and gets any event(s) from it
func GetEvents(ctx *fasthttp.RequestCtx) (ces []j.CloudEvent, mode j.Mode, err error) {
	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	mode = GetMode(ctx)
	switch mode {
	case j.ModeBinary:
		ce, err := CtxBinaryToCE(ctx)
		if err != nil {
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
		if err != nil {
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

// CtxBinaryToCE converts a RequestCtx in Binary mode into a jsonce CloudEvent
func CtxBinaryToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	m := map[string]interface{}{}

	// Required + Optional
	// Note that headers ce-data_base64 and ce-data will be dropped to prevent conflicts
	ctx.Request.Header.VisitAll(func(K, v []byte) {
		k := strings.ToLower(string(K))
		if !strings.HasPrefix(k, "ce-") {
			return
		}
		key := strings.TrimPrefix(k, "ce-")
		if key == "data" || key == "data_base64" {
			err = fmt.Errorf("Binary header forbidden: %s", key)
		}
		m[key] = string(v)
	})
	if err != nil {
		return ce, fmt.Errorf("Could not read binary headers: %s", err.Error())
	}

	// Additional
	j.SetData(m, ctx.PostBody())

	ce = j.CloudEvent{}
	err = ce.FromMap(m)
	return ce, err
}

// CtxStructureJSONToCE converts a RequestCtx in Structured mode with JSON content type into a jsonce CloudEvent
func CtxStructureJSONToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	body := ctx.PostBody()
	ce = j.CloudEvent{}
	err = ce.UnmarshalJSON(body)
	if err != nil {
		return ce, fmt.Errorf("Could not unmarshal to event: %s", err.Error())
	}
	ct := string(ctx.Request.Header.Peek("Content-Type"))
	if len(ce.DataContentType) < 1 {
		ce.DataContentType = ct
	}
	// If both ct and ce.DataContentType were set, this might throw an error according to
	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#311-http-content-type
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

/*
 ██████╗ ██████╗  ██████╗ ██████╗ ██╗   ██╗ ██████╗███████╗
 ██╔══██╗██╔══██╗██╔═══██╗██╔══██╗██║   ██║██╔════╝██╔════╝
 ██████╔╝██████╔╝██║   ██║██║  ██║██║   ██║██║     █████╗
 ██╔═══╝ ██╔══██╗██║   ██║██║  ██║██║   ██║██║     ██╔══╝
 ██║     ██║  ██║╚██████╔╝██████╔╝╚██████╔╝╚██████╗███████╗
 ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═════╝  ╚═════╝  ╚═════╝╚══════╝
*/

// PutEvents determines the mode and content type of a request and puts any event(s) into it
// Note that ces[1...] are dropped unless mode is batch
func PutEvents(ctx *fasthttp.RequestCtx, ces []j.CloudEvent, mode j.Mode) (err error) {
	if len(ces) < 1 {
		return fmt.Errorf("Could not put %d events", len(ces))
	}

	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	switch mode {
	case j.ModeBinary:
		err := CEToCtxBinary(ctx, ces[0])
		if err != nil {
			return fmt.Errorf("Could not set binary event: %s", err.Error())
		}
		return nil
	case j.ModeStructure:
		ce := ces[0]
		ct := ce.DataContentType

		if !strings.HasPrefix(ct, "application/cloudevents+json") {
			return fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ctx.Response.Header.Set("Content-Type", ct)
		err := CEToCtxStructureJSON(ctx, ce)
		if err != nil {
			return fmt.Errorf("Could not set structure event: %s", err.Error())
		}
		return nil
	case j.ModeBatch:
		err = fmt.Errorf("Unimplemented mode: %s", mode)
	default:
		err = fmt.Errorf("Unknown mode: %s", mode)
	}
	return
}

// CEToCtxBinary converts a jsonce CloudEvent into a RequestCtx in Binary mode
func CEToCtxBinary(ctx *fasthttp.RequestCtx, ce j.CloudEvent) (err error) {
	// Required
	ctx.Response.Header.Set("ce-id", ce.Id)
	ctx.Response.Header.Set("ce-source", ce.Source)
	ctx.Response.Header.Set("ce-specversion", ce.SpecVersion)
	ctx.Response.Header.Set("ce-type", ce.Type)
	// Optional
	ctx.Response.Header.Set("Content-Type", ce.DataContentType)
	ctx.Response.Header.Set("ce-dataschema", ce.DataSchema)
	ctx.Response.Header.Set("ce-subject", ce.Subject)
	ctx.Response.Header.Set("ce-time", ce.Time.Format(time.RFC3339Nano))
	// Additional
	ctx.Write(ce.Data)
	for k, v := range ce.Extensions {
		ctx.Response.Header.Set(fmt.Sprintf("ce-%s", k), fmt.Sprintf("%s", v))
	}
	return nil
}

// CEToCtxStructureJSON converts a jsonce CloudEvent into a RequestCtx in Structured mode with JSON content type
func CEToCtxStructureJSON(ctx *fasthttp.RequestCtx, ce j.CloudEvent) (err error) {
	js, err := ce.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Could not marshal event: %s", err.Error())
	}

	ctx.Write(js)
	ctx.Response.Header.Set("Content-Type", ce.DataContentType)

	return nil
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
			ces, mode, err := GetEvents(ctx)
			if err != nil {
				log.Printf("ERR: %s", err.Error())
				ctx.Error(err.Error(), http.StatusBadRequest)
				return
			} else {
				log.Printf("OK : Received %d events in mode %d\n", len(ces), mode)
			}
			fmt.Printf("\tData: %#v\n", ces)
			// ctx.Write([]byte(fmt.Sprintf("STUB:NO REPLY: %#v", ces)))
			PutEvents(ctx, ces, mode)
		}
	}
	if err := fasthttp.ListenAndServe(listenAddr, router); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}
