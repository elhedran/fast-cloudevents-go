# Fast Cloudevents

A small, versatile implementation of the [cloudevents spec](https://github.com/cloudevents/spec) for Go, with support for fasthttp.

[![GoDoc](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go?status.svg)](http://godoc.org/github.com/CreativeCactus/fast-cloudevents-go)
[![Go Report](https://goreportcard.com/badge/github.com/CreativeCactus/fast-cloudevents-go)](https://goreportcard.com/report/github.com/CreativeCactus/fast-cloudevents-go)
[![Sourcegraph](https://sourcegraph.com/github.com/CreativeCactus/fast-cloudevents-go/-/badge.svg)](https://sourcegraph.com/github.com/CreativeCactus/fast-cloudevents-go?badge)

This package exists to replace the heavy, complex, and difficult to use (in my opinion) [go-sdk](https://github.com/cloudevents/sdk-go).

## Example

See [main.go](./main.go)

## Features

- High level getters and setters for fasthttp request/response: [`GetEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#GetEvents), [`SetEvents`](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/fastce#SetEvents)
- [Flexible CloudEvents type](https://godoc.org/github.com/CreativeCactus/fast-cloudevents-go/jsonce#CloudEvent)
- Lightweight, easy to audit

## V1 spec

See [JSON](https://github.com/cloudevents/spec/blob/v1.0/json-format.md), [HTTP](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md) specifications.

### [fasthttp](https://github.com/valyala/fasthttp) support:

- [3.1 Binary Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#31-binary-content-mode) ☑️  Send and receive.
- [3.2 Structured Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#32-structured-content-mode) ☑️  Send and receive.
- [3.3. Batched Content Mode](https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md#33-batched-content-mode) ❌  Not supported yet.

### JSON support:

- [2.2. Type System Mapping](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#22-type-system-mapping) 🕙 Supported on known fields, user must enforce for extensions as needed using type assertion.
- [2.4. JSONSchema Validation](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) ❌  Not tested yet.
- [3. Envelope](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) 🕙 Fully suported, partially complaint.
- [4. JSON Batch Format](https://github.com/cloudevents/spec/blob/v1.0/json-format.md#24-jsonschema-validation) ❌  Not supported yet.

## Conventions

- `cec` means CloudEvent Client
- `ce` and `ces` mean CloudEvent and CloudEvents respectively
- `re` and `res` mean Response CloudEvent and Response CloudEvents respectively

## See also

[FastHTTP best practices](https://github.com/valyala/fasthttp#fasthttp-best-practices).