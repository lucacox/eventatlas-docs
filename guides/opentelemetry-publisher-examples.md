---
post_title: Instrument NATS Publishers for EventAtlas
author1: Luca Cossaro
post_slug: instrument-nats-publishers-for-eventatlas
featured_image: ""
categories:
  - development
tags:
  - eventatlas
  - opentelemetry
  - nats
  - go
  - typescript
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Runnable Go and TypeScript examples that publish NATS messages and export matching OpenTelemetry spans to EventAtlas.
post_date: 2026-08-24
---

# Instrument NATS Publishers for EventAtlas

EventAtlas combines broker declarations with application telemetry. NATS and
JetStream can declare subjects, streams, and consumers, but only the publishing
application can identify the logical service that sent a message.

The runnable examples in this repository demonstrate that missing link:

```text
orders-go-publisher -----------+
                               +--> publishes --> orders.created
orders-typescript-publisher ---+
```

Each program performs two real operations:

1. it publishes a message to the NATS subject `orders.created`;
1. it exports an OpenTelemetry producer span over OTLP/HTTP Protobuf.

EventAtlas normalizes the span into an observation, correlates its destination
with the subject discovered from JetStream, and adds an observed `publishes`
edge to the topology view. The examples use only standard OpenTelemetry and
NATS libraries; applications do not need an EventAtlas SDK.

## Prerequisites

- the sibling `eventatlas`, `eventatlas-web`, and `eventatlas-deploy`
  repositories;
- Docker with Compose;
- Go 1.27 or later for the Go example;
- Node.js 24 and npm for the TypeScript example;
- `curl` and optionally `jq` for API inspection.

Start the local stack and seed the declared NATS topology:

```sh
cd ../eventatlas-deploy
docker compose up --build --detach
./scripts/seed-nats-topology.sh
```

The seed creates the `ORDERS` stream with `orders.created` as one of its
subjects. The local backend discovers NATS every five seconds, exposes its
topology API on port `8080`, and accepts OTLP/HTTP traces on port `4318`.

## Run the Go Publisher

```sh
cd ../eventatlas-docs/examples/otel-go-publisher
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318 go run .
```

The example uses the official NATS Go client and the OpenTelemetry Go
OTLP/HTTP exporter. Its default service identity is
`eventatlas.examples/orders-go-publisher` in the `development` environment.

## Run the TypeScript Publisher

Install the locked dependencies once, then build and run the example:

```sh
cd ../eventatlas-docs/examples/otel-typescript-publisher
npm ci
OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4318 npm start
```

The example uses the official `@nats-io/transport-node` client and the
OpenTelemetry JavaScript OTLP/HTTP Protobuf exporter. Its default service
identity is `eventatlas.examples/orders-typescript-publisher` in the
`development` environment.

## Configuration

Both examples accept the same environment variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `NATS_URL` | `nats://127.0.0.1:4222` | NATS server used for the real publish operation. |
| `OTEL_SERVICE_NAME` | Language-specific publisher name | Logical service name shown by EventAtlas. |
| `EXAMPLE_SERVICE_NAMESPACE` | `eventatlas.examples` | Optional namespace participating in service identity. |
| `EXAMPLE_ENVIRONMENT` | `development` | Must match the environment configured in EventAtlas. |
| `NATS_SUBJECT` | `orders.created` | Physical NATS destination receiving the message. |
| `NATS_SUBJECT_TEMPLATE` | `orders.*` | Low-cardinality logical destination hint. |
| `MESSAGE_BODY` | Demo order event JSON | Message bytes sent to NATS. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | Standard base endpoint used by the OTLP/HTTP exporter. |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | Derived as `/v1/traces` | Optional trace-specific endpoint; include the full path when setting it. |

Exporter headers, timeouts, and compression can also use the standard
OpenTelemetry OTLP environment variables supported by each SDK. In a shared
environment, point applications at an OpenTelemetry Collector and let the
Collector export to EventAtlas. Direct export is used here to keep the example
small and expose the complete data path.

## What the Instrumentation Does

### 1. Identify the service

The tracer provider attaches these resource attributes to every exported span:

| Attribute | EventAtlas meaning |
| --- | --- |
| `service.name` | Required logical service name. |
| `service.namespace` | Optional namespace used in logical identity. |
| `deployment.environment.name` | Environment that must match the EventAtlas runtime. |

Instance IDs, process IDs, pod names, and service versions do not create
separate service nodes. EventAtlas models a logical service, not an individual
process replica.

### 2. Describe the publish operation

The producer span wraps the actual NATS publish and carries:

| Attribute | Example value | Purpose |
| --- | --- | --- |
| `messaging.system` | `nats` | Selects the supported messaging system. |
| `messaging.operation.type` | `send` | Makes the span eligible for the publishing slice. |
| `messaging.destination.name` | `orders.created` | Physical subject used by this message. |
| `messaging.destination.template` | `orders.*` | Preferred low-cardinality correlation hint. |

Use `send` exactly. The current EventAtlas slice deliberately ignores other
operations so it does not double-count instrumentation that emits creation and
sending spans for the same message.

The template is a logical hint, not a separate topology node. EventAtlas first
tries to correlate `orders.*`; because the local fixture declares the concrete
subject `orders.created`, it then falls back to that exact physical name. The
observed edge therefore targets the provider-owned `orders.created` node.

### 3. Confirm the NATS publish

Both examples flush the NATS connection before ending the span. A publish
failure is recorded on the span and returned by the program instead of
reporting a successful observation for work that did not complete.

### 4. Flush telemetry on shutdown

The SDKs use batching. Ending a span only makes it eligible for export; it does
not guarantee that the batch has already left the process. Each example shuts
down its tracer provider before exiting so the final span is exported.

## Verify the Topology

Open the web application at `http://127.0.0.1:5173` or inspect service nodes
through the API:

```sh
curl --silent http://127.0.0.1:8080/api/v1/topology \
  | jq '.nodes[] | select(.kind == "service") | {name, environment}'
```

Inspect the publishing edge produced by the Go example:

```sh
curl --silent http://127.0.0.1:8080/api/v1/topology \
  | jq --arg service orders-go-publisher '
      . as $topology
      | ($topology.nodes | map({key: .id, value: .name}) | from_entries) as $names
      | $topology.edges[]
      | select(.kind == "publishes" and $names[.sourceId] == $service)
      | {source: $names[.sourceId], kind, target: $names[.targetId], evidence}
    '
```

The edge evidence should report mode `observed`, source system
`opentelemetry`, and the observation source configured by the backend.

## Troubleshooting

### The program publishes but no service appears

- Confirm that the OTLP receiver is enabled and reachable on port `4318`.
- Use a base URL for `OTEL_EXPORTER_OTLP_ENDPOINT`; use the full
  `/v1/traces` path only with `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`.
- Make sure the program exits normally so tracer-provider shutdown can flush
  the batch.
- Check backend logs for rejected spans or temporary persistence failures.

### The service appears but the publish edge is unresolved

- Run `eventatlas-deploy/scripts/seed-nats-topology.sh`.
- Wait for the next NATS discovery refresh.
- Confirm that `NATS_SUBJECT` matches a discovered concrete subject or that
  `NATS_SUBJECT_TEMPLATE` uniquely matches a declared destination.

### EventAtlas reports an environment mismatch

`EXAMPLE_ENVIRONMENT` must equal `EVENTATLAS_ENVIRONMENT` on the backend. The
local stack uses `development` for both.

### The span is ignored

The first observation slice accepts only
`messaging.operation.type=send`. Consumer `process` spans and other messaging
operations belong to later slices.

## Privacy Boundary

EventAtlas extracts only the low-cardinality identity fields needed for
topology. It does not retain message payloads, trace IDs, span IDs, message
IDs, arbitrary span attributes, or user identifiers. The raw OTLP request is
discarded after eligible observations have been normalized and persisted.

## References

- [EventAtlas observation architecture](../architecture/observations.md)
- [ADR-0010: Ingest observations from OTLP traces](../adrs/0010-ingest-observations-from-otlp-traces.md)
- [OpenTelemetry Go exporters](https://opentelemetry.io/docs/languages/go/exporters/)
- [OpenTelemetry JavaScript exporters](https://opentelemetry.io/docs/languages/js/exporters/)
- [OTLP exporter configuration](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/)
- [NATS JavaScript client migration](https://github.com/nats-io/nats.js/blob/main/migration.md)
