# Fast Cloudevents

A small, versatile implementation of a superset (relaxed) [cloudevents spec](https://github.com/cloudevents/spec).

This package exists to replace the heavy, complex, and difficult to use [go-sdk](https://github.com/cloudevents/sdk-go).

It is currently a reference implementation of a consumer (server) with fasthttp, but will become a portable set of middlewares.

## V1 spec

https://github.com/cloudevents/spec/blob/v1.0/http-protocol-binding.md

- 3.1 Binary Receive Event :ballot_box_with_check:
- 3.1 Binary Respond Event :negative_squared_cross_mark:
- 3.2 Structured Content Mode :negative_squared_cross_mark:
- 3.3 Batched Content Mode :negative_squared_cross_mark:

