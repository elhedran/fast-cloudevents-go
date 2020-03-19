# Fast Cloudevents

A small and configurable implementation of the [cloudevents spec](https://github.com/cloudevents/spec) for Go, with support for fasthttp.

[![GoDoc](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go?status.svg)](http://godoc.org/github.com/CreativeCactus/fast-cloudevents-go)
[![Go Report](https://goreportcard.com/badge/github.com/CreativeCactus/fast-cloudevents-go)](https://goreportcard.com/report/github.com/CreativeCactus/fast-cloudevents-go)
[![Sourcegraph](https://sourcegraph.com/github.com/CreativeCactus/fast-cloudevents-go/-/badge.svg)](https://sourcegraph.com/github.com/CreativeCactus/fast-cloudevents-go?badge)

This package exists to replace the heavy, complex, and difficult to use (in my opinion) [go-sdk](https://github.com/cloudevents/sdk-go).

## Example

See [main.go](./main.go) and [fastce/examples.go](./fastce/examples.go).

```go
package main

import (
    fastce "github.com/creativecactus/fast-cloudevents-go/fastce"
    jsonce "github.com/creativecactus/fast-cloudevents-go/jsonce"
)

func main(){      
    // An example of a custom unmarshal function
    MyMapToCE := func(cm jsonce.CEMap)(ce jsonce.CloudEvent, err error){
        // In this example, we still want to perform the DefaultCEToMap validation
        // But we will automatically generate an ID if it is not present
        if id, ok := cm["id"].(string); !ok || len(id) < 1 {
            cm["id"] = "SomeRandomRuntimeGeneratedID"
        }
        return jsonce.DefaultMapToCE(cm)
    }

    handler := func(ces jsonce.CloudEvents)(res jsonce.CloudEvents, err error){
        // This is a simple echo server
        res = ces
        return
    }
    CEServer{}.ListenAndServeCE(listenAddr,jsonce.DefaultCEToMap,MyMapToCE,handler)
}
```

## Notes

- While most functions operate on []jsonce.CloudEvent (AKA jsonce.CloudEvents), structured and binary modes will discard all but the first event.
- In binary mode, all extension headers are treated as strings.
This might look strange when sending and receiving in different modes.
To support receiving non-strings in binary mode,
use a custom unmarshal mapper as in the above example.

## Features

- High level server and client types: [`CEClient`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#CEClient), [`CEServer`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#CEServer)
- Mid level getters and setters for fasthttp request/response: [`GetEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#GetEvents), [`SetEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#SetEvents), [`SendEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#SendEvents), [`RecvEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#RecvEvents)
- [Flexible CloudEvents type](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/jsonce#CloudEvent) can be used standalone
- Lightweight, easy to audit, heavily tested
- Good support for CloudEvents spec
- Easy to use

## V1 spec

See [JSON](https://github.com/cloudevents/spec/blob/v1.0/json-format.md), [HTTP](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md) specifications.

### [fasthttp](https://github.com/valyala/fasthttp) support:

- [3.1 Binary Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#31-binary-content-mode) ☑️  Send and receive.
- [3.2 Structured Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#32-structured-content-mode) ☑️  Send and receive.
- [3.3. Batched Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#33-batched-content-mode) ☑️  Send and receive.

### JSON support:

- [2.2. Type System Mapping](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#22-type-system-mapping) ☑️ Supported on known fields, user must enforce for extensions as needed using type assertion.
- [2.4. JSONSchema Validation](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) ❌  Not tested yet.
- [3. Envelope](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) 🕙 Fully suported, partially complaint.
- [4. JSON Batch Format](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) ☑️  Supported.

## Conventions

- `cec` means CloudEvent Client
- `ce` and `ces` mean CloudEvent and CloudEvents respectively
- `res` is sometimes used to refer to a second CloudEvents if there are multiple
- `re` and `res` mean Response CloudEvent and Response CloudEvents respectively
- `Get`/`Set` and `Send`/`Recv` refer to Servers and Clients writing/reading events, respectively

## See also

[FastHTTP best practices](https://github.com/valyala/fasthttp#fasthttp-best-practices).