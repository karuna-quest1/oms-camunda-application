# Worker: update-dashboard

- Worker id (manifest `source.workerId`): `worker:oms-camunda-quest1:com.oms.workers.OmsWorkers#updateDashboard`
- Activity name: `Update-dashboard`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/OmsWorkers.java:169 — `@JobWorker(type = JOB_UPDATE_DASHBOARD)`
- autoComplete: true

## Classification

- Side-effect class: **idempotent-with-key** (confidence 0.95, provenance llm)
- Summary: Updates a dashboard status record for an order, upserting the order's current status and timestamp.
- Idempotency key: `orderId`
- Rationale: The downstream I/O shows a keyed upsert pattern: `update(status.orderId(), ...)` targets a specific row by orderId (the natural key), and if absent, `insert(status.orderId(), ...)` creates it with orderId as the primary key. A concurrent insert race is resolved by falling back to update on DuplicateKeyException. This is a conditional write keyed by orderId: repeating the call with the same orderId and status writes the same record state, making retries safe. The orderId is read from job variables and drives the deduplication mechanism visible in the code.

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
