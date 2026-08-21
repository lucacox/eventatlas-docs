---
post_title: "ADR-0010: Ingest Observations from OTLP Traces"
author1: Luca Cossaro
post_slug: adr-0010-ingest-observations-from-otlp-traces
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - opentelemetry
  - otlp
  - observations
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to ingest messaging observations from standard OTLP trace exports over HTTP and normalize them inside EventAtlas.
post_date: 2026-08-21
---

# ADR-0010: Ingest Observations from OTLP Traces

## Status

Accepted

## Context

EventAtlas already discovers, reconciles, persists, and renders topology
declared by NATS and JetStream. Broker discovery cannot identify the logical
service that publishes to a subject or executes a consumer, so the next
vertical slice must add runtime observations without weakening the distinction
between declared and observed evidence established by ADR-0005 and ADR-0006.

ADR-0006 selected OpenTelemetry as the observation source but deliberately
deferred the transport, Collector deployment, sampling policy, and ingestion
boundary. Applications may already export telemetry through an OpenTelemetry
Collector to one or more observability backends. EventAtlas must fit that
pipeline without requiring an EventAtlas-specific SDK or becoming the system
of record for raw traces.

The OpenTelemetry messaging semantic conventions are still evolving. Resource
identity attributes such as `service.name`, `service.namespace`, and
`service.instance.id` are stable, while the messaging span conventions and
attributes remain in development. Provider-specific NATS conventions are not
currently defined by the standard registry. The adapter therefore needs an
explicit normalization boundary and must not expose raw semantic-convention
keys as domain behavior.

## Decision Drivers

- Accept telemetry from standard OpenTelemetry SDKs and Collectors.
- Avoid coupling instrumented applications directly to EventAtlas.
- Preserve existing trace pipelines and support Collector fan-out.
- Keep OTLP transport and semantic-convention changes outside the core model.
- Start with one small relationship that can be correlated reliably.
- Bound memory, payload, cardinality, and privacy risk at ingestion.
- Retain observations durably without storing raw traces or message payloads.
- Keep the backend as one process with explicit internal adapters.

## Decision

EventAtlas will implement an OTLP trace receiver adapter over HTTP using binary
Protocol Buffers. The adapter will accept `POST /v1/traces` on a dedicated,
configurable listener using the conventional OTLP/HTTP port `4318` in local
development.

The OTLP listener is disabled unless explicitly configured. It is separate
from the Huma API listener so binary telemetry traffic can have independent
request limits, timeouts, authentication, health, and failure handling.
OTLP/gRPC, OTLP/JSON, metrics, and logs are deferred.

An OpenTelemetry Collector is the recommended deployment boundary. Its trace
pipeline may export the same telemetry to EventAtlas and to a tracing backend.
EventAtlas does not replace the Collector or a trace store. It normalizes
eligible messaging spans into topology observations and immediately discards
the raw trace envelope.

The first vertical slice recognizes only messaging send operations and emits:

```text
Service -> publishes -> Destination
```

An eligible span must provide:

- an explicit `service.name` resource attribute that is not an
  `unknown_service` fallback;
- `messaging.operation.type=send`;
- a supported `messaging.system` value;
- `messaging.destination.name`, with optional
  `messaging.destination.template` for low-cardinality correlation;
- a valid span timestamp.

`service.namespace` participates in logical service identity when present.
`service.instance.id`, process identity, pod identity, and service version do
not create separate service nodes. EventAtlas scopes the logical service by
its configured environment. `deployment.environment.name`, when supplied,
must match that environment; when absent, the configured environment is used.

The adapter converts OTLP data into an application-owned observation fact. A
fact carries identity hints, relationship kind, observation time, source, and
an allow-listed metadata set. It never carries trace IDs, span IDs, message
IDs, correlation IDs, payloads, user identifiers, or arbitrary resource and
span attributes.

Destination identity is resolved in the application projection, not in the
OTLP adapter. The first slice correlates a destination hint with exactly one
declared destination in the configured broker scope, preferring
`messaging.destination.template` and falling back to
`messaging.destination.name`. Unresolved or ambiguous facts are retained as
diagnostics but do not create navigable destination nodes in the first slice.

Observations are stored independently from provider snapshots and have a
time-based lifecycle. Repeated facts update `firstSeen`, `lastSeen`, and an
approximate observation count. Active observations participate in a merged
topology view; expired observations do not. Their expiry can never remove or
modify declared evidence.

The receiver follows OTLP response semantics:

- full acceptance returns an OTLP success response;
- partially valid batches return an OTLP partial-success response with the
  number of rejected spans;
- invalid payloads return a non-retryable client error;
- overload and temporary persistence failures return retryable status codes.

Request size, decompressed size, concurrency, and processing time are bounded
and configurable. Authentication and transport security are deployment
concerns, but the listener must support an authentication boundary before any
non-local exposure.

## Consequences

### Positive

- Existing OpenTelemetry SDKs can feed EventAtlas without a custom library.
- A Collector can fan out traces without changing application configuration.
- The domain receives small normalized facts rather than transport payloads.
- EventAtlas does not retain raw traces or high-cardinality identifiers.
- The first slice has one reliable relationship and a narrow acceptance test.
- Separate listeners isolate topology queries from telemetry ingestion load.

### Negative

- EventAtlas must implement the OTLP server response contract and Protobuf
  decoding.
- Trace sampling makes observed topology incomplete by definition.
- Messaging attributes in development require compatibility tests and adapter
  versioning.
- A separate observation store and merged projection add application and
  persistence complexity.
- The first slice cannot yet show consumer-to-service ownership.

### Risks

- Collector retries can deliver duplicate spans. Observation counts are
  approximate and must not be presented as exact traffic metrics.
- Dynamic destination names can create unbounded cardinality. Prefer
  `messaging.destination.template`, enforce length and cardinality limits, and
  reject unsafe values.
- Incorrect service resource configuration can fragment logical services.
  Validate explicit service identity and surface rejected-span diagnostics.
- Semantic-convention changes can break normalization. Keep attribute aliases
  and compatibility behavior in the OTLP adapter and cover them with fixtures.
- Sampling or export failure can make a valid relationship expire. Expiry
  means "not recently observed", never "does not exist".

## Alternatives Considered

### Build an EventAtlas Collector Exporter

A custom Collector exporter could emit normalized facts directly. It was not
selected because users would need a custom Collector distribution or plugin,
and the normalization contract would be split across repositories. It may be
revisited for high-volume deployments after the ingestion model stabilizes.

### Read from a Trace Backend

Querying Jaeger, Tempo, or a commercial tracing platform would reuse stored
traces. It was not selected because it couples EventAtlas to vendor-specific
query APIs, retention, and sampling behavior.

### Derive Topology from Metrics

Collector connectors can aggregate messaging spans into metrics with lower
volume. This was deferred because a portable topology identity contract must
be established first, and aggregated metrics may omit attributes needed for
consumer correlation. Metrics remain a possible optimization and traffic
enrichment source.

### Accept an EventAtlas-specific HTTP Event

A compact custom endpoint would be easy to implement but would require custom
application or Collector integrations and contradict ADR-0006.

### Start with OTLP/gRPC

OTLP/gRPC is standard and efficient. HTTP/protobuf was selected first because
it has a smaller server surface, works over HTTP/1.1, and uses the same
Protobuf data model. gRPC can be added without changing normalized facts.

## References

- [Observation architecture](../architecture/observations.md)
- [ADR-0005: Separate Declared and Observed Topology](0005-separate-declared-and-observed-topology.md)
- [ADR-0006: Treat OpenTelemetry as an Observation Source](0006-treat-opentelemetry-as-observation-source.md)
- [ADR-0007: Start with a Single Backend Process](0007-start-with-single-backend-process.md)
- [OTLP specification](https://opentelemetry.io/docs/specs/otlp/)
- [OTLP exporter specification](https://opentelemetry.io/docs/specs/otel/protocol/exporter/)
- [OpenTelemetry messaging spans](https://opentelemetry.io/docs/specs/semconv/messaging/messaging-spans/)
- [OpenTelemetry service resource conventions](https://opentelemetry.io/docs/specs/semconv/resource/service/)
