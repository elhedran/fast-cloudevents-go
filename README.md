# Fast Cloudevents

A small, versatile implementation of a superset (relaxed) [cloudevents spec](https://github.com/cloudevents/spec).

This package exists to replace the heavy, complex, and difficult to use [go-sdk](https://github.com/cloudevents/sdk-go).

It is currently a reference implementation of a consumer (server) with fasthttp, but will become a portable set of middlewares.

## V1 spec

https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md

### [fasthttp](https://github.com/valyala/fasthttp):

- 3.1 Binary Receive Event ☑️	`FastHTTPToEventBinary`
- 3.1 Binary Respond Event ☑️	`EventToFastHTTPBinary`
- 3.1.3 Metadata Headers ☑️	`FastHTTPToExtensionsBinary` `ExtensionsToFastHTTPBinary` `knownHeader` 
- 3.2 Structured Content Mode 🗷
- 3.3 Batched Content Mode 🗷

