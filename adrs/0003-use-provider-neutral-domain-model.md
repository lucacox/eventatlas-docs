---
post_title: "ADR-0003: Use a Provider-Neutral Domain Model"
author1: Luca Cossaro
post_slug: adr-0003-use-provider-neutral-domain-model
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - domain-model
  - providers
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to keep the EventAtlas domain model provider-neutral.
post_date: 2026-08-16
---

# ADR-0003: Use a Provider-Neutral Domain Model

## Status

Accepted

## Context

EventAtlas will discover and visualize topology from multiple messaging
systems. NATS and JetStream are the first integration, but future integrations
may include Kafka, RabbitMQ, AWS SNS and SQS, Google Pub/Sub, and Azure Service
Bus.

These systems use different concepts and terminology. A NATS subject, Kafka
topic, and RabbitMQ exchange or queue are not identical, but they participate
in comparable messaging relationships. Some provider concepts have no direct
equivalent elsewhere, such as a JetStream stream or an AMQP binding.

If the core model adopts the vocabulary and structure of the first provider,
later integrations will either distort their native semantics or require a
costly redesign of storage, APIs, topology logic, and the user interface.

This decision defines the boundary between the EventAtlas core and messaging
providers. It does not define the persistence schema, public API shape,
provider loading mechanism, or evidence lifecycle.

## Decision Drivers

- Support multiple messaging technologies without redesigning the core.
- Present one coherent topology to API clients and the web application.
- Keep provider SDKs and native types outside domain logic.
- Preserve provider-specific details needed for diagnostics and enrichment.
- Allow topology algorithms and tests to operate independently of a broker.
- Avoid treating NATS and JetStream as universal messaging abstractions.

## Decision

EventAtlas will use a provider-neutral domain model. The core owns stable
messaging concepts, their identities, allowed relationships, and invariants.
Initial core entities include:

- `Service`;
- `Broker`;
- `Destination`;
- `MessagingResource`;
- `Consumer`;
- `Edge` and its supporting `Evidence`;
- `TopologySnapshot`.

`Destination` represents a named messaging address. Provider-specific address
types, such as subject, topic, queue, and exchange, are expressed as
destination kinds rather than separate core entities.

`MessagingResource` represents an optional provider-managed construct involved
in capture, retention, routing, or delivery. Its kind is namespaced, such as
`nats.jetstream.stream`. The core does not require every provider to create an
equivalent resource layer.

Each provider adapter translates native topology into the core model. Native
SDK types, wildcard semantics, configuration objects, and operational details
remain inside the adapter. Details with no stable cross-provider meaning are
retained as namespaced metadata, for example `kafka.partition_count` or
`rabbitmq.exchange_type`.

New core concepts must have clear domain semantics and must be validated
against at least NATS and JetStream, Kafka, and RabbitMQ. A concept used by
only one provider remains a provider-specific kind or metadata until a product
requirement demonstrates that it belongs in the navigable core topology.

## Consequences

### Positive

- Additional providers can reuse topology storage, algorithms, APIs, and UI.
- The first NATS integration cannot define universal domain terminology by
  accident.
- Core logic can be tested with normalized fixtures and no running broker.
- Provider SDK changes are isolated behind adapter boundaries.
- Provider-specific information remains available for diagnostics without
  controlling core behavior.

### Negative

- Every provider requires an explicit normalization layer.
- Some native semantics cannot be represented as first-class core fields.
- Generic terminology can be less familiar than provider-native terminology.
- Cross-provider identity and relationship rules require careful design.
- Evolving the core requires evaluating consequences across several provider
  models, including providers not yet implemented.

### Risks

- The model could collapse into a lowest-common-denominator graph. Preserve
  meaningful resource kinds and relationships instead of flattening every
  provider into generic nodes.
- Provider metadata could become an unstructured substitute for domain
  modeling. Promote a field into the core when it gains stable semantics and a
  demonstrated cross-provider use case.
- Normalization could discard details needed for troubleshooting. Retain
  namespaced metadata and native identifiers at provider boundaries.
- Speculative abstractions could slow the first integration. Add core concepts
  only for concrete topology and product requirements.

## Alternatives Considered

### Model the Core Around NATS and JetStream

This approach would reduce normalization work for the first provider and make
initial implementation more direct. It was not selected because subjects,
streams, and JetStream consumers do not map cleanly to every messaging system.
Generalizing later would change core identities, storage, APIs, and UI
contracts after they were already in use.

### Maintain a Separate Domain Model per Provider

Each provider could expose its native topology with minimal translation. It
was not selected because topology algorithms, persistence, API consumers, and
the UI would need provider-specific branches. A combined view across multiple
messaging systems would also become significantly harder.

### Represent Everything as Untyped Nodes and Edges

A fully generic property graph would accept any provider structure without
changing core types. It was not selected because unconstrained node and edge
labels provide weak semantics, make invariants difficult to enforce, and move
provider branching into every consumer of the graph.

### Use a Fixed Lowest-Common-Denominator Model

The core could expose only services, destinations, publishes, and consumes.
It was not selected because broker-managed resources and routing relationships
are essential to explaining real message paths. Optional typed resources
retain that information without making one provider's topology mandatory.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [ADR-0001: Use Go for the Backend](0001-use-go-for-backend.md)
