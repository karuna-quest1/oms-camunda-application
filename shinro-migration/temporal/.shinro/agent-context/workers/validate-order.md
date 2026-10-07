# Worker: validate-order

- Worker id (manifest `source.workerId`): `worker:oms-camunda-application:com.oms.workers.CommerceWorkers#validateOrder`
- Activity name: `Validate-order`
- Language/SDK: java/spring-zeebe
- Registration: src/main/java/com/oms/workers/CommerceWorkers.java:50 — `@JobWorker(type = JOB_VALIDATE_ORDER, autoComplete = false, maxJobsActive = 150)`
- autoComplete: false (explicit completion)

## Classification

- Side-effect class: **read-only** (confidence 0.95, provenance llm)
- Summary: Validates an order structure in memory, applying any pending corrections before checking business rules, then either throws a business error or completes with the validated order.
- Rationale: The code slice shows only in-memory operations: reading variables, deserializing with ObjectMapper.convertValue, applying corrections via applyCorrection (not shown but appears to be a pure function), calling validationError (a validation check), and logging. No external I/O is visible—no HTTP calls, database writes, or service invocations appear in the delegated call graph (only ObjectMapper.convertValue, Logger.info, Logger.warn). The handler either throws a business error (newThrowErrorCommand) or completes with updated variables (newCompleteCommand), both of which are Zeebe engine operations, not side effects. The comment 'Mock commerce validation is idempotent' reinforces this is a pure validation step. Retrying this worker is safe because it produces the same validation outcome given the same input.

## Migration guidance

- Coverage: **ASSISTED**
- Suggested retry policy: maxAttempts=2147483647
- flag: autoComplete=false + throwError -> typed error branch to user-task

## Variables

Read:

Written:

## Engine API calls (these DISAPPEAR — map to Temporal return patterns)

- `newThrowErrorCommand` (success) at src/main/java/com/oms/workers/CommerceWorkers.java:65 — hint: NonRetryableApplicationError(type="")
- `newCompleteCommand` (success) at src/main/java/com/oms/workers/CommerceWorkers.java:80 — hint: return output, nil

## Source

Read the handler source with read_file: `src/main/java/com/oms/workers/CommerceWorkers.java` (source repo root).
Call graph: ObjectMapper.convertValue, Logger.info, Logger.warn
