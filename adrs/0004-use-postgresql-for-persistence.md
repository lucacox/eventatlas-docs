---
post_title: "ADR-0004: Use PostgreSQL as the Initial Persistence Layer"
author1: Luca Cossaro
post_slug: adr-0004-use-postgresql-for-persistence
featured_image: ""
categories:
  - architecture
tags:
  - adr
  - persistence
  - postgresql
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Decision to use PostgreSQL for EventAtlas durable persistence.
post_date: 2026-08-16
---

# ADR-0004: Use PostgreSQL as the Initial Persistence Layer

## Status

Accepted

## Context

EventAtlas needs durable storage for topology entities, relationships,
provider discovery results, and observed evidence. The stored model must
support reconciliation of provider snapshots, temporal attributes such as
`firstSeen` and `lastSeen`, and queries used by the REST API and web frontend.

Although topology is represented as a graph, the initial domain is composed of
a bounded set of typed entities and relationships. The first product slices do
not require arbitrary-depth graph traversal, graph algorithms, or a query
language exposed directly to users.

EventAtlas should minimize operational dependencies while requirements and
data access patterns are still evolving. Selecting a specialized graph or
time-series database before measuring those patterns would increase deployment
and development complexity without addressing a demonstrated limitation.

This decision selects the initial durable database. It does not define the
relational schema, migration library, retention policy, repository interfaces,
or API query model.

## Decision Drivers

- Transactional reconciliation of topology snapshots and evidence.
- Strong constraints for typed entities, identities, and relationships.
- Sufficient query support for the initial topology API and filters.
- Flexible storage for namespaced provider metadata.
- Mature Go drivers, migration tools, and operational practices.
- One primary database for the initial deployment.
- A clear path to indexing and query optimization as usage becomes measurable.

## Decision

EventAtlas will use PostgreSQL as its initial durable persistence layer.
Topology entities, edges, observations, and discovery state will be represented
using relational structures with explicit keys and constraints.

Provider-specific metadata may use PostgreSQL `jsonb` where its shape is not
part of the stable core model. Core identity, relationship semantics, source,
and lifecycle fields must remain queryable typed columns rather than being
hidden in generic JSON documents.

Schema changes will use versioned migrations. Persistence will be accessed
through application-owned ports so that SQL and PostgreSQL-specific details do
not enter the domain model.

An in-memory implementation may be used for tests, local demonstrations, and
the first read-only vertical slice. It is not a second production persistence
strategy and does not replace PostgreSQL when durable history or reconciliation
is required.

No graph database, time-series database, or search engine will be added until
measured product requirements demonstrate that PostgreSQL cannot meet a
specific workload with reasonable schema design and indexing.

## Consequences

### Positive

- Snapshot reconciliation can update related topology facts atomically.
- Foreign keys and constraints can enforce identity and relationship rules.
- SQL supports the initial lookup, filtering, aggregation, and join patterns.
- `jsonb` preserves provider-specific metadata without expanding the core
  schema for every native setting.
- The project operates one durable database during the MVP.
- PostgreSQL has mature backup, migration, observability, and Go integration
  tooling.

### Negative

- Graph traversals require joins or recursive common table expressions rather
  than a graph-native query language.
- Relational schemas and migrations add work while the domain model evolves.
- `jsonb` fields require discipline to avoid becoming an unstructured storage
  default.
- Running PostgreSQL adds more local setup than an embedded database.
- High-volume raw telemetry may eventually require different retention or
  storage strategies.

### Risks

- Poor edge and identity indexes could make topology queries slow. Capture
  representative query plans and add indexes based on measured access paths.
- Provider metadata could hide fields needed by core queries. Promote fields
  with stable domain meaning into typed columns through migrations.
- Raw observations could grow without bound. Define aggregation and retention
  policies before ingesting production telemetry at scale.
- Persistence concerns could leak into the domain model. Keep mapping and SQL
  behavior inside storage adapters and test domain logic independently.

## Alternatives Considered

### Neo4j or Another Graph Database

A graph database provides native traversal and graph query capabilities. It
was not selected because the initial topology has typed, predictable
relationships and bounded API queries that PostgreSQL can represent directly.
Introducing a graph database would add another operational model before a
graph-specific workload has demonstrated its value.

### Document Database

A document database would make provider metadata and evolving records easy to
store. It was not selected because EventAtlas relies on relationships,
identity constraints, reconciliation, and cross-entity queries that fit a
relational database more directly.

### SQLite

SQLite would provide a lightweight embedded option for local use and early
prototypes. It was not selected as the primary durable store because the
target deployment needs a shared server database with established concurrency,
backup, and operational patterns. An in-memory or local adapter can still
support isolated tests and demonstrations.

### In-Memory Storage Only

In-memory storage minimizes the first implementation effort and is useful for
a read-only vertical slice. It was not selected as the durable strategy because
topology history, observed evidence, and reconciliation state must survive
process restarts.

### Specialized Time-Series Database

A time-series database could optimize high-volume metrics and retention. It
was not selected because the initial requirement is topology persistence with
limited aggregated evidence, not raw telemetry storage. This decision can be
revisited if measured ingestion and query workloads exceed PostgreSQL's role.

## References

- [EventAtlas domain model](../architecture/domain-model.md)
- [PostgreSQL documentation](https://www.postgresql.org/docs/)
- [PostgreSQL JSON types](https://www.postgresql.org/docs/current/datatype-json.html)
