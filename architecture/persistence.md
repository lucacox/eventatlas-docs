---
post_title: EventAtlas Topology Persistence
author1: Luca Cossaro
post_slug: eventatlas-topology-persistence
featured_image: ""
categories:
  - architecture
tags:
  - persistence
  - postgresql
  - topology
  - reconciliation
ai_note: AI-assisted and reviewed by Luca Cossaro.
summary: Durable storage and reconciliation semantics for EventAtlas topology snapshots.
post_date: 2026-08-20
---

# EventAtlas Topology Persistence

## Purpose

EventAtlas persists the latest successfully reconciled topology for each
discovery source and scope. It also persists runtime observation aggregates in
a separate relational boundary. Persistence lets the API restore declared
topology after a process restart and lets recent observations retain their
time range and approximate count across restarts.

PostgreSQL adapters sit behind the application-owned topology and observation
store ports. SQL types, migration state, and database lifecycle do not enter
the topology or observation cores.

## Current Boundary

The declared boundary stores one current topology per pair:

```text
sourceId + discovery scope
```

An EventAtlas process currently exposes one configured pair through
`GET /api/v1/topology`. The relational key is source-scoped so that later
versions can retain several provider and observation sources without changing
their identities.

The observation boundary stores one aggregate per identity defined in
[Observation architecture](observations.md):

```text
source + scope + relationship + service key + destination hint
```

Snapshot replacement never modifies observation aggregates. Observation
upserts never modify provider-owned snapshot rows.

The current runtime still does not implement:

- historical topology browsing;
- retention policies for raw snapshots;
- OpenTelemetry ingestion;
- API exposure of the merged topology view;
- PostgreSQL-specific query endpoints.

## Reconciliation Semantics

Discovery produces an immutable incoming snapshot. The application layer
reconciles it with the current snapshot before persistence so every store
adapter has identical behavior.

### Full snapshots

A full snapshot is authoritative for its exact source and scope.

- Nodes and edges absent from the incoming snapshot are removed.
- Incoming nodes and edges replace facts with the same identity.
- Matching evidence keeps the earliest `firstSeen` and the latest `lastSeen`.
- Provider metadata from the incoming evidence is retained.

### Partial snapshots

A partial snapshot contains trustworthy facts but is not authoritative about
absence.

- Incoming nodes and edges are inserted or updated.
- Existing nodes and edges absent from the incoming snapshot are retained.
- Matching evidence merges its temporal range as for a full snapshot.
- Partial errors and completeness remain visible on the reconciled snapshot.

Snapshots from a different source or scope must never be reconciled together.

## Transaction Boundary

Replacing the current topology is one PostgreSQL transaction:

1. remove the previous relational representation for the same source and
   scope;
2. write snapshot metadata;
3. write typed nodes;
4. write directed edges;
5. write ordered evidence records;
6. commit atomically.

Readers see either the previous complete topology or the new complete
topology. They never observe a partially written graph. A failed transaction
leaves the previous snapshot available.

An observation batch is also one PostgreSQL transaction. Every normalized fact
is validated before the transaction begins. Each fact then inserts or updates
its aggregate, retaining the earliest `firstSeen`, latest `lastSeen`, metadata
associated with the latest observation, and a saturating approximate count. A
failure rolls back the complete batch.

## Relational Model

The initial schema contains:

| Table | Responsibility |
| --- | --- |
| `topology_snapshots` | Current snapshot identity, source, scope, capture time, completeness, errors, cursor, and metadata. |
| `topology_nodes` | Stable node identity, core kind-specific columns, display name, and provider attributes. |
| `topology_edges` | Directed typed relationship and deterministic position in the snapshot. |
| `topology_edge_evidence` | Evidence source, mode, source system, time range, metadata, and deterministic position. |
| `topology_observations` | Runtime fact identity, earliest and latest observation times, approximate count, and allow-listed metadata. |
| `eventatlas_schema_migrations` | Applied versioned database migrations. |

Core identity and relationship fields use typed relational columns.
Provider-specific attributes and metadata use `jsonb` objects. Partial errors
use a JSON array because they are snapshot diagnostics rather than query
identity.

Foreign keys and checks enforce:

- node and edge kinds recognized by the current schema;
- edge endpoints belonging to the same source-scoped snapshot;
- evidence time ranges where `lastSeen >= firstSeen`;
- observation identity fields and time ranges where `lastSeen >= firstSeen`;
- positive observation counts;
- JSON object and array shapes;
- cascading cleanup when a source-scoped snapshot is replaced.

The Go domain constructors remain the final validation boundary when data is
rehydrated. A database row that cannot recreate a valid domain object is a
storage error, not a partially usable topology.

## Startup and Failure Behavior

When PostgreSQL is configured, EventAtlas runs migrations and loads the current
snapshot before the first discovery attempt.

- If discovery succeeds, the reconciled result becomes current.
- If discovery fails but a persisted snapshot exists, the API starts with that
  snapshot and periodic refresh continues retrying.
- If neither discovery nor persisted topology is available, startup fails.
- A later failed refresh never replaces the last valid topology.

The in-memory adapter remains available for tests and demonstrations when no
database URL is configured. It is not the durable production strategy.

## Acceptance Criteria

The persistence slice is complete when:

- a full graph round-trips through PostgreSQL without losing typed fields,
  ordering, evidence, or provider metadata;
- full and partial reconciliation behavior is covered by tests;
- a replacement failure leaves the previous snapshot readable;
- a backend restart can serve the last persisted topology before a successful
  provider refresh;
- migrations and PostgreSQL integration tests run reproducibly in Docker;
- the public topology API remains unchanged.

## References

- [Domain model](domain-model.md)
- [ADR-0004: Use PostgreSQL](../adrs/0004-use-postgresql-for-persistence.md)
- [ADR-0005: Separate declared and observed topology](../adrs/0005-separate-declared-and-observed-topology.md)
- [Observation architecture](observations.md)
