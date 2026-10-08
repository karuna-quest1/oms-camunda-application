# Worker: update-dashboard

- Worker id (manifest `source.workerId`): `worker:oms-camunda-application:com.oms.workers.OmsWorkers#updateDashboard`
- Activity name: `Update-dashboard`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:169 — `@JobWorker(type = JOB_UPDATE_DASHBOARD)`
- autoComplete: true

## Classification

- Side-effect class: **idempotent-with-key** (confidence 0.95, provenance llm)
- Summary: Updates a dashboard with order status using an upsert pattern keyed by orderId
- Idempotency key: `orderId`
- Rationale: The code slice shows JdbcDashboardRepository.upsert performs a keyed write using orderId as the primary key. The implementation follows update-or-insert logic: first attempts UPDATE WHERE orderId matches, on 0 rows affected tries INSERT (which will fail on duplicate key if concurrent), then falls back to UPDATE. This is a proper upsert pattern with orderId as the deduplication key — multiple executions with the same orderId will overwrite the same row with the latest status and timestamp, making retries safe.

## Migration guidance

- Coverage: **ASSISTED**

## Variables

Read:

Written:

## Callee source — READ THESE TOO

The handler delegates its real work to the code below. Read each file, follow it
further if it delegates again (service → client → http), and translate that logic
into the activity body — do NOT stub a success return.


## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/OmsWorkers.java` (source repo root).
Call graph: Logger.error, Logger.info, DashboardRepository.upsert, Logger.warn
