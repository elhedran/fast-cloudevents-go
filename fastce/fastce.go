package fastce

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	j "github.com/creativecactus/fast-cloudevents-go/jsonce"

	"github.com/valyala/fasthttp"
)

/*
 ████████╗██╗     ██████╗ ██████╗
 ╚══██╔══╝██║     ██╔══██╗██╔══██╗
    ██║   ██║     ██║  ██║██████╔╝
    ██║   ██║     ██║  ██║██╔══██╗
    ██║   ███████╗██████╔╝██║  ██║
    ╚═╝   ╚══════╝╚═════╝ ╚═╝  ╚═╝
*/

// CEServer is too high level to maintain, and does not offer a good abstraction
// Instead, use SetEventsCtx and GetEventsCtx directly, or GetEvents and SetEvents
// // CEServer is a convenience wrapper around GetEvents[Ctx] and SetEvents[Ctx]
// // It can be instantiated literally using a fasthttp.RequestCtx
// // but makes no guarantees about the state of the underlying RequestCtx
// type CEServer struct {
// 	Ctx *fasthttp.RequestCtx
// }

// // SetEvents is a convenience wrapper around the public function of the same name
// func (cesrv CEServer) SetEvents(ces []j.CloudEvent, mode j.Mode) (err error) {
// 	return SetEventsCtx(cesrv.Ctx, ces, mode)
// }

// // GetEvents is a convenience wrapper around the public function of the same name
// func (cesrv CEServer) GetEvents() (ces []j.CloudEvent, mode j.Mode, err error) {
// 	return GetEventsCtx(cesrv.Ctx)
// }

// CEClient is a convenience wrapper around SendEvents and RecvEvents
type CEClient struct {
	Request  *fasthttp.Request
	Response *fasthttp.Response
	Released bool
	Client   *fasthttp.HostClient
}

// NewCEClient creates a CEClient for a given URI and method
func NewCEClient(method, URLString string) (cec CEClient, err error) {
	URL, err := url.Parse(URLString)
	if err != nil {
		err = fmt.Errorf("Could not parse %s as URL (scheme://host:port/path): %s", URLString, err.Error())
		return
	}

	if host := URL.Hostname(); j.InSlice(host, []string{
		"",
		"::",
		"[::]",
		"0.0.0.0",
	}) {
		if port := URL.Port(); len(port) > 0 {
			URL.Host = fmt.Sprintf("localhost:%s", port)
		} else {
			URL.Host = "localhost"
		}
	}

	c := &fasthttp.HostClient{
		Addr: URL.Host,
	}
	c.DialDualStack = true // Allow IPv6
	c.MaxConns = 10

	cec = CEClient{
		Request:  fasthttp.AcquireRequest(),
		Response: fasthttp.AcquireResponse(),
		Released: false,
		Client:   c,
	}

	cec.Request.SetRequestURI(URL.String())
	cec.Request.Header.SetMethod(method)
	return
}

// Send performs the underlying request of the CEClient
// It should be called after calling .SendCE
func (cec *CEClient) Send() (err error) {
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 100

	err = cec.Client.DoTimeout(cec.Request, cec.Response, 30*time.Second)
	if err != nil {
		err = fmt.Errorf("HTTP Error: %s", err.Error())
	}
	return err
}

// Release must be called to GC the request and response
func (cec *CEClient) Release() {
	if !cec.Released {
		cec.Released = false
		fasthttp.ReleaseRequest(cec.Request)
		fasthttp.ReleaseResponse(cec.Response)
	}
}

// SendEvents allows sending CloudEvents to the server
func (cec *CEClient) SendEvents(ces []j.CloudEvent, mode j.Mode) error {
	return SendEvents(cec.Request, ces, mode)
}

// RecvEvents allows receiving CloudEvents in the server response
func (cec *CEClient) RecvEvents() (ces []j.CloudEvent, mode j.Mode, err error) {
	return RecvEvents(cec.Response)
}

// ToCtx type functionality cannot be supported because
// the intuitions of RequestCtx do not carry in this case:
// The reference to the response held by the client would be detatched.
// // ToCtx presents the client as a fasthttp.RequestCtx
// func (cec *CEClient) ToCtx() *fasthttp.RequestCtx {
// 	return &fasthttp.RequestCtx{
// 		Request: *cec.Request,
// 		Response: cec.Response,
// 	}
// }

/*
 ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗      ██╗███╗   ██╗
 ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗     ██║████╗  ██║
 ███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝     ██║██╔██╗ ██║
 ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗     ██║██║╚██╗██║
 ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║     ██║██║ ╚████║
 ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝     ╚═╝╚═╝  ╚═══╝

*/

// GetEventsCtx receives cloudevents from a Request in any mode
func GetEventsCtx(ctx *fasthttp.RequestCtx) (ces []j.CloudEvent, mode j.Mode, err error) {
	return GetEvents(&ctx.Request)
}

// GetEvents receives cloudevents from a Request in any mode
func GetEvents(req *fasthttp.Request) (ces []j.CloudEvent, mode j.Mode, err error) {
	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	rr := ReqResFromReq(req)

	if mode, err = rr.GetMode(); err != nil {
		err = fmt.Errorf("Could not get mode: %s", err.Error())
		return
	}

	switch mode {
	case j.ModeBinary:
		ce, err := rr.BinaryToCE()
		if err != nil {
			return ces, mode, fmt.Errorf("Could not get binary event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeStructure:
		// Determine the media type with which to parse the event
		// Or reject anything other than JSON
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
		ct := string(req.Header.Peek("Content-Type"))
		if !strings.HasPrefix(ct, "application/cloudevents+json") {
			return ces, mode, fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ce, err := rr.StructureJSONToCE()
		if err != nil {
			return ces, mode, fmt.Errorf("Could not get structure event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeBatch:
		// Determine the media type with which to parse the events
		// Or reject anything other than JSON
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
		ct := string(req.Header.Peek("Content-Type"))
		if !strings.HasPrefix(ct, "application/cloudevents-batch+json") {
			return ces, mode, fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ces, err = rr.BatchJSONToCE()
		if err != nil {
			err = fmt.Errorf("Could not get batch events: %s", err.Error())
		}
		return ces, mode, err
	default:
		err = fmt.Errorf("Unknown mode: %d", mode)
	}
	return
}

/*
 ███████╗███████╗██████╗ ██╗   ██╗███████╗██████╗       ██████╗ ██╗   ██╗████████╗
 ██╔════╝██╔════╝██╔══██╗██║   ██║██╔════╝██╔══██╗     ██╔═══██╗██║   ██║╚══██╔══╝
 ███████╗█████╗  ██████╔╝██║   ██║█████╗  ██████╔╝     ██║   ██║██║   ██║   ██║
 ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██╔══╝  ██╔══██╗     ██║   ██║██║   ██║   ██║
 ███████║███████╗██║  ██║ ╚████╔╝ ███████╗██║  ██║     ╚██████╔╝╚██████╔╝   ██║
 ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═╝      ╚═════╝  ╚═════╝    ╚═╝
*/

// SetEventsCtx accepts the mode and content of a response and puts any event(s) into it
// Note that ces[1...] are dropped unless mode is batch
func SetEventsCtx(ctx *fasthttp.RequestCtx, ces []j.CloudEvent, mode j.Mode) (err error) {
	return SetEvents(&ctx.Response, ces, mode)
}

// SetEvents accepts the mode and content of a response and puts any event(s) into it
// Note that ces[1...] are dropped unless mode is batch
func SetEvents(res *fasthttp.Response, ces []j.CloudEvent, mode j.Mode) (err error) {
	if len(ces) < 1 {
		return fmt.Errorf("Could not put %d events", len(ces))
	}

	rr := ReqResFromRes(res)
	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	switch mode {
	case j.ModeBinary:
		err := rr.CEToBinary(ces[0])
		if err != nil {
			return fmt.Errorf("Could not set binary event: %s", err.Error())
		}
		return nil
	case j.ModeStructure:
		err := rr.CEToStructureJSON(ces[0])
		if err != nil {
			return fmt.Errorf("Could not set structure event: %s", err.Error())
		}
		return nil
	case j.ModeBatch:
		err := rr.CEToBatchJSON(ces)
		if err != nil {
			return fmt.Errorf("Could not set batch events: %s", err.Error())
		}
		return nil
	default:
		err = fmt.Errorf("Unknown mode: %d", mode)
	}
	return
}

/*
  ██████╗██╗     ██╗███████╗███╗   ██╗████████╗      ██████╗ ██╗   ██╗████████╗
 ██╔════╝██║     ██║██╔════╝████╗  ██║╚══██╔══╝     ██╔═══██╗██║   ██║╚══██╔══╝
 ██║     ██║     ██║█████╗  ██╔██╗ ██║   ██║        ██║   ██║██║   ██║   ██║
 ██║     ██║     ██║██╔══╝  ██║╚██╗██║   ██║        ██║   ██║██║   ██║   ██║
 ╚██████╗███████╗██║███████╗██║ ╚████║   ██║        ╚██████╔╝╚██████╔╝   ██║
  ╚═════╝╚══════╝╚═╝╚══════╝╚═╝  ╚═══╝   ╚═╝         ╚═════╝  ╚═════╝    ╚═╝
*/

// return  accepts the mode and content of a request and puts any event(s) into it
// Note that this does not perform the request, see CEClient class for that
// Note that ces[1...] are dropped unless mode is batch
func SendEvents(req *fasthttp.Request, ces []j.CloudEvent, mode j.Mode) (err error) {
	if len(ces) < 1 {
		return fmt.Errorf("Could not put %d events", len(ces))
	}

	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#13-content-modes
	switch mode {
	case j.ModeBinary:
		err := CEToRequestBinary(req, ces[0])
		if err != nil {
			return fmt.Errorf("Could not send binary event: %s", err.Error())
		}
		return nil
	case j.ModeStructure:
		err := CEToRequestStructureJSON(req, ces[0])
		if err != nil {
			return fmt.Errorf("Could not send structure event: %s", err.Error())
		}
		return nil
	case j.ModeBatch:
		err := CEToRequestBatchJSON(req, ces)
		if err != nil {
			return fmt.Errorf("Could not send batch events: %s", err.Error())
		}
		return nil
	default:
		err = fmt.Errorf("Unknown mode: %d", mode)
	}
	return
}

/*
  ██████╗██╗     ██╗███████╗███╗   ██╗████████╗     ██╗███╗   ██╗
 ██╔════╝██║     ██║██╔════╝████╗  ██║╚══██╔══╝     ██║████╗  ██║
 ██║     ██║     ██║█████╗  ██╔██╗ ██║   ██║        ██║██╔██╗ ██║
 ██║     ██║     ██║██╔══╝  ██║╚██╗██║   ██║        ██║██║╚██╗██║
 ╚██████╗███████╗██║███████╗██║ ╚████║   ██║        ██║██║ ╚████║
  ╚═════╝╚══════╝╚═╝╚══════╝╚═╝  ╚═══╝   ╚═╝        ╚═╝╚═╝  ╚═══╝
*/

// RecvEvents receives cloudevents from a Response in any mode
func RecvEvents(res *fasthttp.Response) (ces []j.CloudEvent, mode j.Mode, err error) {
	if mode, err = ReqResFromRes(res).GetMode(); err != nil {
		err = fmt.Errorf("Could not get mode: %s", err.Error())
		return
	}

	switch mode {
	case j.ModeBinary:
		ce, err := ResponseBinaryToCE(res)
		if err != nil {
			return ces, mode, fmt.Errorf("Could not receive binary event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeStructure:
		// Determine the media type with which to parse the event
		// Or reject anything other than JSON
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
		ct := string(res.Header.Peek("Content-Type"))
		if !strings.HasPrefix(ct, "application/cloudevents+json") {
			return ces, mode, fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ce, err := ResponseStructureJSONToCE(res)
		if err != nil {
			return ces, mode, fmt.Errorf("Could not receive structure event: %s", err.Error())
		}
		ces = append(ces, ce)
		return ces, mode, nil
	case j.ModeBatch:
		// Determine the media type with which to parse the events
		// Or reject anything other than JSON
		// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
		ct := string(res.Header.Peek("Content-Type"))
		if !strings.HasPrefix(ct, "application/cloudevents-batch+json") {
			return ces, mode, fmt.Errorf("Unknown event content media type: %s", ct)
		}

		ces, err := ResponseBatchJSONToCE(res)
		if err != nil {
			return ces, mode, fmt.Errorf("Could not receive batch events: %s", err.Error())
		}
		return ces, mode, nil
	default:
		err = fmt.Errorf("Unknown mode: %d", mode)
	}
	return

}

/*
  ██████╗ ███████╗███╗   ██╗███████╗██████╗ ██╗ ██████╗
 ██╔════╝ ██╔════╝████╗  ██║██╔════╝██╔══██╗██║██╔════╝
 ██║  ███╗█████╗  ██╔██╗ ██║█████╗  ██████╔╝██║██║
 ██║   ██║██╔══╝  ██║╚██╗██║██╔══╝  ██╔══██╗██║██║
 ╚██████╔╝███████╗██║ ╚████║███████╗██║  ██║██║╚██████╗
  ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝╚═╝ ╚═════╝
*/

type req *fasthttp.Request

// IsReqRes is an explicit guarantee that the underlying type is
// consistent with a fasthttp.Request/Response type,
// such as .Header
// func(req)IsReqRes(){}

type res *fasthttp.Response

// IsReqRes is an explicit guarantee that the underlying type is
// consistent with a fasthttp.Request/Response type,
// such as .Header
// func(res)IsReqRes(){}

// ReqRes represents either a fasthttp.Request or fasthttp.Response
// Interfaces cannot be used conveniently because of property dependence
type ReqRes struct {
	r interface{} // Underlying request or response
	// AppendBody([]byte)
	// Body() []byte
	// IsReqRes()
}

func ReqResFromReq(req *fasthttp.Request) ReqRes {
	return ReqRes{
		r: req,
	}

}
func ReqResFromRes(res *fasthttp.Response) ReqRes {
	return ReqRes{
		r: res,
	}
}

func (rr ReqRes) Body() (p []byte, err error) {
	switch v := rr.r.(type) {
	case *fasthttp.Request:
		p = v.Body()
	case *fasthttp.Response:
		p = v.Body()
	case fasthttp.Request:
		p = v.Body()
	case fasthttp.Response:
		p = v.Body()
	default:
		err = fmt.Errorf("Body: Invalid ReqRes type: %T", v)
	}
	return p, err
}

func (rr ReqRes) AppendBody(p []byte) (err error) {
	switch v := rr.r.(type) {
	case *fasthttp.Request:
		v.AppendBody(p)
	case *fasthttp.Response:
		v.AppendBody(p)
	case fasthttp.Request:
		v.AppendBody(p)
	case fasthttp.Response:
		v.AppendBody(p)
	default:
		err = fmt.Errorf("AppendBody: Invalid ReqRes type: %T", v)
	}
	return err
}

type RRHeader interface {
	Header() []byte
	IsHTTP11() bool
	Len() int
	Peek(key string) []byte
	PeekBytes(key []byte) []byte
	Set(key, value string)
	SetBytesK(key []byte, value string)
	SetBytesKV(key, value []byte)
	SetBytesV(key string, value []byte)
	String() string
	VisitAll(f func(key, value []byte))
}

func (rr ReqRes) Header() (RRHeader, error) {
	switch v := rr.r.(type) {
	case *fasthttp.Request:
		return &v.Header, nil
	case *fasthttp.Response:
		return &v.Header, nil
	case fasthttp.Request:
		return &v.Header, nil
	case fasthttp.Response:
		return &v.Header, nil
	default:
		return nil, fmt.Errorf("Header: Invalid ReqRes type: %T", v)
	}
}

// Writing CE //

// CEToBinary puts a CloudEvent into a Request or Response in Binary mode
func (rr ReqRes) CEToBinary(ce j.CloudEvent) (err error) {
	head, err := rr.Header()
	if err != nil {
		return fmt.Errorf("Could not access Header: %s", err.Error())
	}
	// Required
	head.Set("ce-id", ce.Id)
	head.Set("ce-source", ce.Source)
	head.Set("ce-specversion", ce.SpecVersion)
	head.Set("ce-type", ce.Type)
	// Optional
	head.Set("Content-Type", ce.DataContentType)
	head.Set("ce-dataschema", ce.DataSchema)
	head.Set("ce-subject", ce.Subject)
	head.Set("ce-time", ce.Time.Format(time.RFC3339Nano))
	// Additional
	rr.AppendBody(ce.Data)
	for k, v := range ce.Extensions {
		data, err := json.Marshal(v)
		if err != nil { 
			return fmt.Errorf("Could not marshal Header: %s: %s", k, err.Error())
		}
		head.Set(fmt.Sprintf("ce-%s", k), string(data))
	}
	return nil
}

// CEToStructureJSON puts a CloudEvent into a Request or Response in JSON in Structure mode
func (rr ReqRes) CEToStructureJSON(ce j.CloudEvent) (err error) {
	js, err := ce.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Could not marshal event: %s", err.Error())
	}

	head, err := rr.Header()
	if err != nil {
		return fmt.Errorf("Could not get Header: %s", err.Error())
	}

	rr.AppendBody(js)
	head.Set("Content-Type", "application/cloudevents+json")
	return nil
}

// CEToBatchJSON puts a CloudEvent into a Request or Response in JSON in Batch mode
func (rr ReqRes) CEToBatchJSON(ces []j.CloudEvent) (err error) {
	js, err := json.Marshal(ces)
	if err != nil {
		return fmt.Errorf("Could not marshal event: %s", err.Error())
	}

	head, err := rr.Header()
	if err != nil {
		return fmt.Errorf("Could not get Header: %s", err.Error())
	}

	rr.AppendBody(js)
	head.Set("Content-Type", "application/cloudevents-batch+json")

	return nil
}

// Reading CE //

// BinaryToCE reads a Request or Response in Binary mode into CloudEvents
func (rr ReqRes) BinaryToCE() (ce j.CloudEvent, err error) {
	m := map[string]interface{}{}

	head, err := rr.Header()
	if err != nil {
		return ce, fmt.Errorf("Could not get Header: %s", err.Error())
	}

	// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#311-http-content-type
	dct := string(head.Peek("ce-datacontenttype"))
	if len(dct) > 0 {
		return ce, fmt.Errorf("Expected empty ce-datacontenttype to be empty, got: %s", dct)
	}

	// Required + Optional
	// Note that headers ce-data_base64 and ce-data will be dropped to prevent conflicts
	head.VisitAll(func(K, v []byte) {
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

	ct := string(head.Peek("Content-Type"))
	ce.DataContentType = ct

	// Additional
	body, err := rr.Body()
	if err != nil {
		return ce, fmt.Errorf("Could not read body: %s", err.Error())
	}
	j.SetData(m, body)

	ce = j.CloudEvent{}
	err = ce.FromMap(m)
	return ce, err
}

// StructureJSONToCE reads a Request or Response in JSON in Structure mode into CloudEvents
func (rr ReqRes) StructureJSONToCE() (ce j.CloudEvent, err error) {
	body, err := rr.Body()
	if err != nil {
		return ce, fmt.Errorf("Could not read body: %s", err.Error())
	}
	ce = j.CloudEvent{}
	err = ce.UnmarshalJSON(body)
	if err != nil {
		return ce, fmt.Errorf("Could not unmarshal to event: %s", err.Error())
	}
	return ce, err
}

// BatchJSONToCE reads a Request or Response in JSON in Batch mode into CloudEvents
func (rr ReqRes) BatchJSONToCE() (ces []j.CloudEvent, err error) {
	body, err := rr.Body()
	if err != nil {
		return ces, fmt.Errorf("Could not read body: %s", err.Error())
	}

	ces = []j.CloudEvent{}
	err = json.Unmarshal(body, &ces)
	if err != nil {
		return ces, fmt.Errorf("Could not unmarshal to events: %s", err.Error())
	}
	return ces, err
}

/*
 ██╗   ██╗████████╗██╗██╗     ██╗████████╗██╗███████╗███████╗
 ██║   ██║╚══██╔══╝██║██║     ██║╚══██╔══╝██║██╔════╝██╔════╝
 ██║   ██║   ██║   ██║██║     ██║   ██║   ██║█████╗  ███████╗
 ██║   ██║   ██║   ██║██║     ██║   ██║   ██║██╔══╝  ╚════██║
 ╚██████╔╝   ██║   ██║███████╗██║   ██║   ██║███████╗███████║
  ╚═════╝    ╚═╝   ╚═╝╚══════╝╚═╝   ╚═╝   ╚═╝╚══════╝╚══════╝
*/

// GetMode uses the Content Type header to determine the content mode of the request
// https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#3-http-message-mapping
func (rr ReqRes) GetMode() (mode j.Mode, err error) {
	head, err := rr.Header()
	if err != nil {
		err = fmt.Errorf("Could not get Header: %s", err.Error())
		return
	}
	ct := string(head.Peek("Content-Type"))

	mode = j.ModeBinary
	if strings.HasPrefix(ct, "application/cloudevents-batch") {
		mode = j.ModeBatch
	} else if strings.HasPrefix(ct, "application/cloudevents") {
		mode = j.ModeStructure
	}
	return
}

/*
██╗    ██╗██████╗  █████╗ ██████╗ ██████╗ ███████╗██████╗ ███████╗
██║    ██║██╔══██╗██╔══██╗██╔══██╗██╔══██╗██╔════╝██╔══██╗██╔════╝
██║ █╗ ██║██████╔╝███████║██████╔╝██████╔╝█████╗  ██████╔╝███████╗
██║███╗██║██╔══██╗██╔══██║██╔═══╝ ██╔═══╝ ██╔══╝  ██╔══██╗╚════██║
╚███╔███╔╝██║  ██║██║  ██║██║     ██║     ███████╗██║  ██║███████║
 ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝     ╚══════╝╚═╝  ╚═╝╚══════╝
*/

// READ REQUEST

func RequestBinaryToCE(req *fasthttp.Request) (ce j.CloudEvent, err error) {
	if ce, err = ReqResFromReq(req).BinaryToCE(); err != nil {
		err = fmt.Errorf("Read Request Error: %s", err.Error())
	}
	return
}

func RequestStructureJSONToCE(req *fasthttp.Request) (ce j.CloudEvent, err error) {
	if ce, err = ReqResFromReq(req).StructureJSONToCE(); err != nil {
		err = fmt.Errorf("Read Request Error: %s", err.Error())
	}
	return
}

func RequestBatchJSONToCE(req *fasthttp.Request) (ces []j.CloudEvent, err error) {
	if ces, err = ReqResFromReq(req).BatchJSONToCE(); err != nil {
		err = fmt.Errorf("Read Request Error: %s", err.Error())
	}
	return
}

// READ RESPONSE

func ResponseBinaryToCE(res *fasthttp.Response) (ce j.CloudEvent, err error) {
	if ce, err = ReqResFromRes(res).BinaryToCE(); err != nil {
		err = fmt.Errorf("Read Response Error: %s", err.Error())
	}
	return
}

func ResponseStructureJSONToCE(res *fasthttp.Response) (ce j.CloudEvent, err error) {
	if ce, err = ReqResFromRes(res).StructureJSONToCE(); err != nil {
		err = fmt.Errorf("Read Response Error: %s", err.Error())
	}
	return
}

func ResponseBatchJSONToCE(res *fasthttp.Response) (ces []j.CloudEvent, err error) {
	if ces, err = ReqResFromRes(res).BatchJSONToCE(); err != nil {
		err = fmt.Errorf("Read Response Error: %s", err.Error())
	}
	return
}

// WRITE REQUEST

// CEToRequestBinary applies a jsonce CloudEvent to a given Request in Binary mode
func CEToRequestBinary(req *fasthttp.Request, ce j.CloudEvent) (err error) {
	if err := ReqResFromReq(req).CEToBinary(ce); err != nil {
		err = fmt.Errorf("Write Request Error: %s", err.Error())
	}
	return
}

// CEToRequestStructureJSON applies a jsonce CloudEvent to a given Request in JSON in Structured mode
func CEToRequestStructureJSON(req *fasthttp.Request, ce j.CloudEvent) (err error) {
	if err := ReqResFromReq(req).CEToStructureJSON(ce); err != nil {
		err = fmt.Errorf("Write Request Error: %s", err.Error())
	}
	return
}

// CEToRequestBatchJSON applies a jsonce CloudEvent to a given Request in JSON in batched mode
func CEToRequestBatchJSON(req *fasthttp.Request, ces []j.CloudEvent) (err error) {
	if err := ReqResFromReq(req).CEToBatchJSON(ces); err != nil {
		err = fmt.Errorf("Write Request Error: %s", err.Error())
	}
	return
}

// WRITE RESPONSE

// CEToResponseBinary applies a jsonce CloudEvent to a Response in Binary mode
func CEToResponseBinary(res *fasthttp.Response, ce j.CloudEvent) (err error) {
	if err := ReqResFromRes(res).CEToBinary(ce); err != nil {
		err = fmt.Errorf("Write Response Error: %s", err.Error())
	}
	return
}

// CEToResponseStructureJSON applies a jsonce CloudEvent to a Response in Structured mode with JSON content type
func CEToResponseStructureJSON(res *fasthttp.Response, ce j.CloudEvent) (err error) {
	if err := ReqResFromRes(res).CEToStructureJSON(ce); err != nil {
		err = fmt.Errorf("Write Response Error: %s", err.Error())
	}
	return
}

// CEToResponseBatchJSON applies a jsonce CloudEvent to a Response in batched mode with JSON content type
func CEToResponseBatchJSON(res *fasthttp.Response, ces []j.CloudEvent) (err error) {
	if err := ReqResFromRes(res).CEToBatchJSON(ces); err != nil {
		err = fmt.Errorf("Write Response Error: %s", err.Error())
	}
	return
}

// DEPRECATING...

// CtxBinaryToCE reads a RequestCtx in Binary mode into a jsonce CloudEvent
// Deprecation candidate
func CtxBinaryToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	return RequestBinaryToCE(&ctx.Request)
}

// CtxStructureJSONToCE reads a RequestCtx in Structured mode with JSON content type into a jsonce CloudEvent
// Deprecation candidate
func CtxStructureJSONToCE(ctx *fasthttp.RequestCtx) (ce j.CloudEvent, err error) {
	return RequestStructureJSONToCE(&ctx.Request)
}

// CtxBatchJSONToCE reads a RequestCtx in batched mode with JSON content type into jsonce CloudEvents
// Deprecation candidate
func CtxBatchJSONToCE(ctx *fasthttp.RequestCtx) (ces []j.CloudEvent, err error) {
	return RequestBatchJSONToCE(&ctx.Request)
}

// CEToCtxBinary applies a jsonce CloudEvent to a RequestCtx in Binary mode
// Deprecation candidate
func CEToCtxBinary(ctx *fasthttp.RequestCtx, ce j.CloudEvent) (err error) {
	return CEToResponseBinary(&ctx.Response, ce)
}

// CEToCtxStructureJSON applies a jsonce CloudEvent to a RequestCtx in Structured mode with JSON content type
// Deprecation candidate
func CEToCtxStructureJSON(ctx *fasthttp.RequestCtx, ce j.CloudEvent) (err error) {
	return CEToResponseStructureJSON(&ctx.Response, ce)
}

// CEToCtxBatchJSON applies a jsonce CloudEvent to a RequestCtx in batched mode with JSON content type
// Deprecation candidate
func CEToCtxBatchJSON(ctx *fasthttp.RequestCtx, ces []j.CloudEvent) (err error) {
	return CEToResponseBatchJSON(&ctx.Response, ces)
}

// PutEvents is a deprecated alias of SetEvents
// Deprecation candidate
func PutEvents(ctx *fasthttp.RequestCtx, ces []j.CloudEvent, mode j.Mode) (err error) {
	fmt.Printf("WARN: PutEvents is deprecated, use SetEvents.\n")
	return SetEvents(&ctx.Response, ces, mode)
}
