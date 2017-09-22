Extremely proof-of-concept-level zipkin-to-Honeycomb forwarder. Speaks the
zipkin V1 API: send your spans to `/api/v1/spans`.

Next steps:

- thrift support
- support passthrough to an actual zipkin instance
- tests
- stdout output for debugging
- retry logic
- handle annotations
- handle endpoints in [binary]Annotations

