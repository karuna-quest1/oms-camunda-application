# Worker: update-dashboard

- Activity name: `Update-dashboard`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:169 — `@JobWorker(type = JOB_UPDATE_DASHBOARD)`
- autoComplete: true

## Classification

- Side-effect class: **idempotent-with-key** (confidence 0.95, provenance llm)
- Summary: Updates a dashboard with the current order status using an upsert operation keyed by orderId
- Idempotency key: `orderId`
- Rationale: The code slice shows a two-phase upsert (update-then-insert-or-update) keyed by orderId. The terminal I/O in JdbcDashboardRepository.upsert performs UPDATE WHERE orderId = ?, and on zero rows affected, attempts INSERT (protected by primary key constraint) with fallback to UPDATE on DuplicateKeyException. This is a keyed write pattern: orderId gates the mutation, and replaying with the same orderId produces the same final state (last-write-wins semantics). The operation is retry-safe because orderId uniquely identifies the row, making subsequent executions overwrite the same record rather than create duplicates.

## Migration guidance

- Coverage: **ASSISTED**

## Variables

Read:

Written:

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/OmsWorkers.java` (source repo root).
Call graph: Logger.error, Logger.info, DashboardRepository.upsert, Logger.warn
