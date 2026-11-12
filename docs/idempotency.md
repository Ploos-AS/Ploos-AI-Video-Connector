# Idempotency keys (PR #14)

Every new local video job receives a random 128-bit hexadecimal ID. This ID is also stored in SQLite as the `idempotency_key` **before** the connector contacts an upstream provider. The asynchronous HTTP adapter sends this value in the `Idempotency-Key` header on `POST /jobs`.

**Upstream contract:** The provider must enforce that repeated POSTs with the same key and request parameters resolve to the same remote job, not a second charge. Merely accepting the header is insufficient. Confirm this behavior in each provider before enabling any retry.

**Current safety behavior:** The connector still **does not automatically retry** an ambiguous upstream POST, including after a crash with no persisted remote ID. A stored key alone cannot guarantee no duplicate billing if the upstream ignores idempotency. On restart, known remote IDs are polled; unknown IDs fail closed. The key is groundwork for future safe reconciliation and retries, not proof of exactly-once execution.

The synchronous HTTP provider does not implement the idempotency interface. SQLite must be configured for durable key storage. Provider-specific key retention periods and conflict semantics must be documented before commercial use.
